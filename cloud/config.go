// Package cloud berisi implementasi SaaS Control Plane: autentikasi multi-tenant,
// rate limiting berbasis plan, usage tracking, dan REST API untuk billing.
package cloud

import (
	"os"

	// go-gateway/config adalah module core Gateway Realtime (open source).
	// SaaS mengimpor config core lalu menambahkan env SaaS seperti database URL
	// dan Stripe keys tanpa memodifikasi module core.
	"go-gateway/config"
)

// Config menyimpan seluruh konfigurasi runtime: config core dari gateway_realtime
// ditambah env SaaS (PostgreSQL, Stripe) yang hanya relevan untuk cloud binary.
type Config struct {
	Core                config.Config // Konfigurasi core: port, Redis, JWT, origin, log level.
	DatabaseURL         string        // PostgreSQL connection string (wajib untuk SaaS).
	StripeSecretKey     string        // Stripe secret key untuk memproses subscription & payment.
	StripeWebhookSecret string        // Stripe webhook signing secret untuk verifikasi event.
}

// Load membaca env variables dan menggabungkan config core dengan config SaaS.
// Fungsi ini dipanggil sekali saat startup di cmd/server/main.go.
func Load() Config {
	core := config.Load()
	return Config{
		Core:                core,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}
