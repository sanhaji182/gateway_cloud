# Gateway Cloud — SaaS Control Plane

> 🇬🇧 [English](./README.md)


> **🤖 Dibangun dengan AI** — Project ini dikembangkan dengan bantuan AI code generation sebagai alat bantu, di bawah arahan dan review developer manusia. Setiap baris kode telah diperiksa dan diuji sebelum masuk ke production.
> **Private repository.** Proprietary. Not open source.
> Lihat `../gateway/SaaS_ARCHITECTURE.md` untuk blueprint arsitektur lengkap.

SaaS layer di atas [Gateway Realtime](https://github.com/sanhaji182/gateway_realtime) — core open-source diimpor sebagai Go module via replace directive.

## Arsitektur

```
Browser/Backend
      │
      ▼
┌──────────────────────────────────────┐
│  Gateway Cloud (binary ini)          │
│  ┌────────────────────────────────┐  │
│  │  Core Gateway (go-gateway)     │  │ ← Open Source
│  │  WebSocket / Health / Auth     │  │
│  └──────────┬─────────────────────┘  │
│             │ Extension Points       │
│  ┌──────────▼─────────────────────┐  │
│  │  SaaS Extensions               │  │ ← Proprietary
│  │  • TenantAuthenticator         │  │
│  │  • PlanRateLimiter             │  │
│  │  • UsageTracker                │  │
│  └──────────┬─────────────────────┘  │
│             │                        │
│  ┌──────────▼─────────────────────┐  │
│  │  SaaS API                      │  │
│  │  POST /api/cloud/register      │  │
│  │  GET  /api/cloud/usage         │  │
│  │  GET  /api/cloud/tenant        │  │
│  │  POST /api/cloud/stripe/webhook│  │
│  └────────────────────────────────┘  │
└──────────┬───────────────────────────┘
           │
    ┌──────┴──────┐
    │  Redis      │  │  PostgreSQL  │
    │  pub/sub    │  │  tenants     │
    │  rate limit │  │  usage_events│
    └─────────────┘  └─────────────┘
```

## Struktur

```
gateway_cloud/            # PRIVATE
├── cmd/server/main.go    # Cloud binary entrypoint — inject SaaS extensions ke core
├── cloud/                # SaaS implementations
│   ├── api.go            # REST endpoints (register, usage, tenant, Stripe webhook)
│   ├── config.go         # SaaS config loader (core + Stripe + Database)
│   ├── db.go             # PostgreSQL connection pool (MustDB)
│   ├── migrate.go        # Auto-migration (tenants, usage_events, indexes)
│   ├── ratelimit.go      # Plan-based rate limiting (Redis token bucket)
│   ├── tenant.go         # Multi-tenant: TenantAuthenticator, CreateTenant, PlanLimits
│   └── usage.go          # Event tracking — EventHook → batch insert usage_events
├── web/                  # Next.js SaaS frontend (coming soon)
├── worker/               # Background jobs (coming soon)
├── Dockerfile.cloud      # Multi-stage Go build
├── docker-compose.prod.yml # PostgreSQL + Redis + gateway-cloud
└── go.mod                # → replace go-gateway => ../gateway/backend_go
```

## Quick Start

```bash
# 1. Start semua services
docker compose -f docker-compose.prod.yml up -d

# 2. Register tenant — dapat API key
curl -X POST http://localhost:4001/api/cloud/register \
  -H "Content-Type: application/json" \
  -d '{"name": "My SaaS", "email": "dev@example.com"}'
# → {"tenant":{...}, "api_key":"pk_abc123..."}

# 3. Cek usage tenant
curl "http://localhost:4001/api/cloud/usage?tenant_id=<uuid>&period=24h"

# 4. Publish event via WebSocket (pakai X-Tenant-Key header)
# (WebSocket connection dengan header tenant)
```

## Plan Tiers

| Tier | Events/Menit | Koneksi | Harga (coming soon) |
|---|---|---|---|
| **Free** | 100 | 5 | Gratis |
| **Pro** | 10,000 | 1,000 | $/bulan |
| **Enterprise** | 100,000 | 10,000 | Custom |

## Build

```bash
go build -o gateway-cloud ./cmd/server
```

## Verifikasi

```bash
# Build harus sukses
go build -o gateway-cloud ./cmd/server

# Binary jalan dengan --help (akan gagal karena env kosong — itu normal)
./gateway-cloud
# → DATABASE_URL is required — set PostgreSQL connection string
```

## Environment Variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL (tenants, usage_events) |
| `REDIS_URL` | Yes | — | Redis (pub/sub + rate limit) |
| `JWT_SECRET` | Yes | — | HMAC signing (min 64 chars) |
| `ALLOWED_ORIGINS` | Yes | — | CORS origins (comma-separated) |
| `PORT` | No | 4001 | HTTP listen port |
| `LOG_LEVEL` | No | info | zerolog level |
| `STRIPE_SECRET_KEY` | No | — | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | No | — | Stripe webhook signing secret |

## Extension Points

SaaS tidak memodifikasi core Gateway. Semua integrasi lewat 3 interface di `go-gateway/extensions`:

```go
// Di cmd/server/main.go:
mux.Handle("/ws", handler.WSHandler{
    Auth:        tenantAuth,    // TenantAuthenticator
    RateLimiter: planLimiter,   // PlanRateLimiter
    EventHook:   usageTracker,  // UsageTracker
})
```

Core Gateway tetap bisa di-upgrade tanpa konflik — selama interface tidak berubah.

## Author

> **Dibangun oleh [Sonick Sanhaji](https://www.linkedin.com/in/sansanhaji/)** — Software developer. Arsitektur dan review oleh manusia. Eksekusi dibantu AI.

## Next Steps

- [ ] Next.js SaaS frontend di `web/` — signup, login, dashboard
- [ ] Stripe integration penuh — webhook verify + update plan
- [ ] Stripe Customer Portal — billing self-service
- [ ] Email notification (welcome email, usage warning, payment receipt)
- [ ] Monitoring dashboard SaaS (Grafana)
- [ ] Private GitHub repo → push commit ini
