package cloud

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// API menyimpan dependency untuk SaaS REST endpoint.
// Endpoint SaaS (register, usage, tenant, stripe) di-mount oleh cmd/server/main.go
// pada HTTP mux yang sama dengan core Gateway handler.
type API struct {
	DB  *sql.DB       // Koneksi PostgreSQL untuk query tenant dan usage.
	Cfg Config        // Konfigurasi SaaS (termasuk Stripe keys).
	Log zerolog.Logger // Logger dengan module "cloud_api" untuk tracing.
}

// NewAPI membuat API instance dengan logger bertag "cloud_api".
// Dipanggil sekali di cmd/server/main.go saat startup.
func NewAPI(db *sql.DB, cfg Config) *API {
	return &API{DB: db, Cfg: cfg, Log: log.With().Str("module", "cloud_api").Logger()}
}

// Register menangani POST /api/cloud/register
//
// Request:  {"name": "My Project", "email": "user@example.com"}
// Response: {"tenant": {...}, "api_key": "pk_..."}
//
// Endpoint ini mendaftarkan tenant baru di tabel tenants dan mengembalikan
// API key yang harus dipakai sebagai header X-Tenant-Key di semua request.
// Email dicatat untuk komunikasi (welcome email, billing notification) —
// untuk v1 disimpan di log saja, next step integrasi email provider.
func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	// Preflight CORS — browser kirim OPTIONS sebelum POST cross-origin.
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name and email required"})
		return
	}
	// Buat tenant baru — plan default "free".
	tenant, err := CreateTenant(a.DB, body.Name)
	if err != nil {
		a.Log.Error().Err(err).Str("email", body.Email).Msg("register tenant failed")
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "registration failed"})
		return
	}
	a.Log.Info().Str("tenant_id", tenant.ID).Str("email", body.Email).Msg("tenant registered")
	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant":  tenant,
		"api_key": tenant.APIKey,
	})
}

// Usage menangani GET /api/cloud/usage?tenant_id=xxx&period=24h
//
// Mengembalikan statistik usage untuk satu tenant dalam periode tertentu.
// Period mendukung format Go duration string: 1h, 24h, 168h (7 hari), 720h (30 hari).
//
// Response:
//
//	{
//	  "usage": { "total_publishes": 1234, "total_subscribes": 567, "total_connects": 89, "payload_bytes": 1024000 },
//	  "period": "24h"
//	}
func (a *API) Usage(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tenant_id required"})
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h" // Default: 24 jam terakhir.
	}
	dur, err := time.ParseDuration(period)
	if err != nil {
		dur = 24 * time.Hour
	}
	since := time.Now().Add(-dur)

	// Agregasi pakai FILTER clause PostgreSQL — lebih efisien dari multiple queries.
	var stats struct {
		TotalPublishes  int64 `json:"total_publishes"`
		TotalSubscribes int64 `json:"total_subscribes"`
		TotalConnects   int64 `json:"total_connects"`
		PayloadBytes    int64 `json:"payload_bytes"`
	}
	a.DB.QueryRowContext(r.Context(),
		`SELECT
			COUNT(*) FILTER (WHERE event_type = 'publish'),
			COUNT(*) FILTER (WHERE event_type = 'subscribe'),
			COUNT(*) FILTER (WHERE event_type = 'connect'),
			COALESCE(SUM(payload_bytes), 0)
		FROM usage_events WHERE tenant_id = $1 AND created_at > $2`,
		tenantID, since,
	).Scan(&stats.TotalPublishes, &stats.TotalSubscribes, &stats.TotalConnects, &stats.PayloadBytes)
	writeJSON(w, http.StatusOK, map[string]any{"usage": stats, "period": period})
}

// GetTenant menangani GET /api/cloud/tenant?api_key=pk_...
//
// Mengembalikan detail tenant berdasarkan API key.
// Dipakai untuk verifikasi kredensial dan dashboard SaaS customer.
func (a *API) GetTenant(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "api_key required"})
		return
	}
	tenant, err := GetTenantByKey(a.DB, apiKey)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenant": tenant})
}

// StripeWebhook menangani POST /api/cloud/stripe/webhook
//
// Menerima event dari Stripe (checkout.session.completed, customer.subscription.updated, dll)
// untuk mengupdate plan tenant di database.
//
// TODO next phase:
//   - Verifikasi Stripe webhook signature (Stripe-Signature header)
//   - Parse event type → update tenants.plan sesuai subscription
//   - Kirim email notifikasi upgrade/downgrade
func (a *API) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Untuk v1, cukup acknowledge event — full implementation di next iteration.
	writeJSON(w, http.StatusOK, map[string]any{"received": true})
}

// setCORS memasang header CORS untuk endpoint SaaS.
// Wildcard (*) dipakai karena endpoint SaaS tidak pakai cookie-based auth.
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Key, X-User-ID")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

// writeJSON menulis response JSON dengan status HTTP eksplisit.
// Helper untuk menjaga response format konsisten di semua endpoint SaaS.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
