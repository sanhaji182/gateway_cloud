package cloud

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
)

// RunMigrations menjalankan migrasi database secara idempoten saat startup.
// Semua statement pakai IF NOT EXISTS agar aman dijalankan berulang kali
// tanpa error — penting untuk rolling deployment dan auto-recovery.
//
// Migrasi mencakup:
//   1. tenants — metadata tenant beserta plan tier (free/pro/enterprise)
//   2. usage_events — log setiap event lifecycle untuk billing per-tenant
//   3. Indexes — idx_usage_tenant_date + idx_usage_type untuk query analytics cepat
func RunMigrations(db *sql.DB, log zerolog.Logger) {
	migrations := []string{
		// Tenants: setiap tenant punya plan tier yang menentukan batas rate limit.
		// api_key dibuat unik dan dipakai sebagai kredensial publish dari backend customer.
		`CREATE TABLE IF NOT EXISTS tenants (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name        TEXT NOT NULL,
			plan        TEXT NOT NULL DEFAULT 'free',
			api_key     TEXT NOT NULL UNIQUE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		// Usage events: dicatat oleh UsageTracker (implementasi EventHook).
		// Tabel ini untuk billing, usage analytics, dan monitoring per tenant.
		`CREATE TABLE IF NOT EXISTS usage_events (
			id            BIGSERIAL PRIMARY KEY,
			tenant_id     UUID NOT NULL REFERENCES tenants(id),
			event_type    TEXT NOT NULL,
			channel       TEXT NOT NULL DEFAULT '',
			payload_bytes BIGINT NOT NULL DEFAULT 0,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		// Index untuk query usage per tenant dalam rentang waktu tertentu.
		// Digunakan oleh endpoint GET /api/cloud/usage.
		`CREATE INDEX IF NOT EXISTS idx_usage_tenant_date ON usage_events(tenant_id, created_at)`,

		// Index untuk agregasi event per tipe (publish, subscribe, connect).
		// Mempercepat dashboard analytics SaaS.
		`CREATE INDEX IF NOT EXISTS idx_usage_type ON usage_events(event_type, created_at)`,
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatal().Err(err).Int("migration", i+1).Msg("migration failed — database may be unreachable or schema conflict")
		}
	}
	fmt.Printf("SaaS migrations %d/%d applied\n", len(migrations), len(migrations))
}
