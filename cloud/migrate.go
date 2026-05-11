package cloud

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
)

// RunMigrations menjalankan migration schema PostgreSQL saat startup.
// Menggunakan IF NOT EXISTS agar idempoten dan aman dijalankan ulang.
func RunMigrations(db *sql.DB, log zerolog.Logger) {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name        TEXT NOT NULL,
			plan        TEXT NOT NULL DEFAULT 'free',
			api_key     TEXT NOT NULL UNIQUE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS usage_events (
			id            BIGSERIAL PRIMARY KEY,
			tenant_id     UUID NOT NULL REFERENCES tenants(id),
			event_type    TEXT NOT NULL,
			channel       TEXT NOT NULL DEFAULT '',
			payload_bytes BIGINT NOT NULL DEFAULT 0,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_tenant_date ON usage_events(tenant_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_type ON usage_events(event_type, created_at)`,
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatal().Err(err).Int("migration", i+1).Msg("migration failed")
		}
	}
	fmt.Printf("SaaS migrations %d/3 applied\n", len(migrations))
}
