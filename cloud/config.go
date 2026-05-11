package cloud

import (
	"os"

	"go-gateway/config"
)

// Load menggabungkan config core milik gateway_realtime dengan env SaaS tambahan.
func Load() Config {
	core := config.Load()
	return Config{
		Core:                core,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}

type Config struct {
	Core                config.Config
	DatabaseURL         string
	StripeSecretKey     string
	StripeWebhookSecret string
}
