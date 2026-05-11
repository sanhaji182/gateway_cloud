package cloud

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// Tenant adalah data tenant SaaS beserta plan tier-nya.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"` // free, pro, enterprise
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantAuthenticator mengimplementasikan extensions.Authenticator.
type TenantAuthenticator struct {
	DB *sql.DB
}

func (a TenantAuthenticator) Authenticate(r *http.Request) (userID, tenantID string, ok bool) {
	tenantKey := r.Header.Get("X-Tenant-Key")
	userID = r.Header.Get("X-User-ID")
	if tenantKey == "" || userID == "" {
		return "", "", false
	}
	var t Tenant
	err := a.DB.QueryRowContext(r.Context(),
		"SELECT id, name, plan FROM tenants WHERE api_key = $1",
		tenantKey,
	).Scan(&t.ID, &t.Name, &t.Plan)
	if err != nil {
		return "", "", false
	}
	return userID, t.ID, true
}

// GetTenantByKey mengambil tenant berdasarkan API key.
func GetTenantByKey(db *sql.DB, apiKey string) (*Tenant, error) {
	var t Tenant
	err := db.QueryRowContext(context.Background(),
		"SELECT id, name, plan, api_key, created_at FROM tenants WHERE api_key = $1",
		apiKey,
	).Scan(&t.ID, &t.Name, &t.Plan, &t.APIKey, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateTenant mendaftarkan tenant baru dengan API key unik.
func CreateTenant(db *sql.DB, name string) (*Tenant, error) {
	apiKey := "pk_" + genToken(16)
	var t Tenant
	err := db.QueryRow(
		"INSERT INTO tenants (name, plan, api_key) VALUES ($1, $2, $3) RETURNING id, name, plan, api_key, created_at",
		name, "free", apiKey,
	).Scan(&t.ID, &t.Name, &t.Plan, &t.APIKey, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	return &t, nil
}

func genToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// PlanLimits mengembalikan batasan per tier.
func PlanLimits(plan string) (eventsPerMinute int, connections int) {
	switch plan {
	case "enterprise":
		return 100_000, 10_000
	case "pro":
		return 10_000, 1_000
	default: // free
		return 100, 5
	}
}
