package cloud

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type API struct {
	DB  *sql.DB
	Cfg Config
	Log zerolog.Logger
}

func NewAPI(db *sql.DB, cfg Config) *API {
	return &API{DB: db, Cfg: cfg, Log: log.With().Str("module", "cloud_api").Logger()}
}

// Register — POST /api/cloud/register
// Body: {"name": "My Project", "email": "user@example.com"}
// Response: {"tenant": {...}, "api_key": "pk_..."}
func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
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
	tenant, err := CreateTenant(a.DB, body.Name)
	if err != nil {
		a.Log.Error().Err(err).Msg("register tenant failed")
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "registration failed"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant":  tenant,
		"api_key": tenant.APIKey,
	})
}

// Usage — GET /api/cloud/usage?tenant_id=xxx&period=24h
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
		period = "24h"
	}
	dur, err := time.ParseDuration(period)
	if err != nil {
		dur = 24 * time.Hour
	}
	since := time.Now().Add(-dur)

	var stats struct {
		TotalPublishes int64 `json:"total_publishes"`
		TotalSubscribes int64 `json:"total_subscribes"`
		TotalConnects   int64 `json:"total_connects"`
		PayloadBytes    int64 `json:"payload_bytes"`
	}
	row := a.DB.QueryRowContext(r.Context(),
		`SELECT
			COUNT(*) FILTER (WHERE event_type = 'publish'),
			COUNT(*) FILTER (WHERE event_type = 'subscribe'),
			COUNT(*) FILTER (WHERE event_type = 'connect'),
			COALESCE(SUM(payload_bytes), 0)
		FROM usage_events WHERE tenant_id = $1 AND created_at > $2`,
		tenantID, since,
	)
	row.Scan(&stats.TotalPublishes, &stats.TotalSubscribes, &stats.TotalConnects, &stats.PayloadBytes)
	writeJSON(w, http.StatusOK, map[string]any{"usage": stats, "period": period})
}

// GetTenant — GET /api/cloud/tenant?api_key=pk_...
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

// StripeWebhook — POST /api/cloud/stripe/webhook
func (a *API) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// TODO: verifikasi Stripe signature + update tenant plan
	writeJSON(w, http.StatusOK, map[string]any{"received": true})
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Key, X-User-ID")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
