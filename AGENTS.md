# Agent Instructions — Gateway Cloud (SaaS Control Plane)

> **Private repository.** Proprietary. Baca `../AGENTS.md` ROOT dulu untuk konteks integrasi.
> Core open source ada di `../gateway/` — BOLEH baca, JANGAN ubah kode core.

---

## Project Structure

```
gateway_cloud/
├── cmd/server/main.go    # Cloud binary entrypoint — inject SaaS extensions ke core
├── cloud/
│   ├── api.go            # SaaS REST endpoints (register, usage, tenant, stripe webhook)
│   ├── config.go         # Config loader: core + SaaS env (Stripe, Database)
│   ├── db.go             # PostgreSQL connection pool (MustDB)
│   ├── migrate.go        # Auto-migration: tenants & usage_events tables + indexes
│   ├── ratelimit.go      # Plan-based rate limiting (Redis INCR + EXPIRE)
│   ├── tenant.go         # Multi-tenant: TenantAuthenticator, CreateTenant, PlanLimits
│   └── usage.go          # UsageTracker: EventHook → async batch insert → PostgreSQL
├── Dockerfile.cloud      # Multi-stage Go build
├── docker-compose.prod.yml # PostgreSQL + Redis + gateway-cloud
└── go.mod                # replace go-gateway => ../gateway/backend_go
```

## Dependencies

Go module mengimpor core via replace directive:
```
require go-gateway v0.1.0
replace go-gateway => ../gateway/backend_go
```

Dependencies eksternal:
- `github.com/lib/pq` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis client
- `github.com/rs/zerolog` — Structured logger

## Build, Test, Run

```bash
# Build
go build -o gateway-cloud ./cmd/server

# Binary start (perlu env variables)
DATABASE_URL=postgres://gateway:password@localhost:5432/gateway_cloud?sslmode=disable \
REDIS_URL=redis://localhost:6379 \
JWT_SECRET=dev-secret-change-in-production \
ALLOWED_ORIGINS=http://localhost:3000 \
./gateway-cloud
```

## Extension Points

Core Gateway (`../gateway/backend_go/extensions/extensions.go`) menyediakan 3 interface:

1. **Authenticator** — `Authenticate(r *http.Request) (userID, tenantID string, ok bool)`
2. **RateLimiter** — `Allow(tenantID, key string, limit int) bool`
3. **EventHook** — `OnConnect/OnDisconnect/OnSubscribe/OnUnsubscribe/OnPublish(...)`

SaaS binary menginject implementations melalui struct di handler:
```go
mux.Handle("/ws", handler.WSHandler{
    Auth:        tenantAuth,    // cloud.TenantAuthenticator
    RateLimiter: planLimiter,   // cloud.PlanRateLimiter
    EventHook:   usageTracker,  // cloud.UsageTracker
})
```

## Database

PostgreSQL dengan auto-migration (`cloud/migrate.go`). Dua tabel:

- **tenants** — id (UUID), name, plan (free/pro/enterprise), api_key (pk_...), created_at
- **usage_events** — id (BIGSERIAL), tenant_id (FK), event_type, channel, payload_bytes, created_at

## SaaS API

| Method | Path | Function |
|---|---|---|
| POST | /api/cloud/register | Register tenant baru |
| GET | /api/cloud/usage?tenant_id=X&period=24h | Usage stats |
| GET | /api/cloud/tenant?api_key=pk_... | Detail tenant |
| POST | /api/cloud/stripe/webhook | Stripe events |

## Coding Style

- Go 1.22, standard library style
- Doc comments bahasa Inggris (package, fungsi, struct)
- Shell comments untuk penjelasan logic
- File maksimal ~200 baris; split jika melebihi
- Error handling eksplisit — jangan abaikan error return
