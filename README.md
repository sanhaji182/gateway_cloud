# Gateway Cloud — SaaS Control Plane

> **Private repository.** Proprietary. Not open source.

SaaS layer di atas [Gateway Realtime](https://github.com/sanhaji182/gateway_realtime) — core open-source diimpor sebagai Go module.

## Struktur

```
gateway_cloud/            # PRIVATE
├── cmd/server/main.go    # Cloud binary entrypoint
├── cloud/                # SaaS implementations
│   ├── api.go            # REST endpoints (register, usage, tenant)
│   ├── config.go         # SaaS config loader
│   ├── db.go             # PostgreSQL connection
│   ├── migrate.go        # Auto-migration
│   ├── ratelimit.go      # Plan-based rate limiting
│   ├── tenant.go         # Multi-tenant auth
│   └── usage.go          # Event tracking → billing
├── web/                  # Next.js SaaS frontend (coming soon)
├── worker/               # Background jobs (coming soon)
└── go.mod                # → replace go-gateway => ../gateway/backend_go
```

## Quick Start

```bash
# 1. Install dependencies
docker compose -f docker-compose.prod.yml up -d

# 2. Register a tenant
curl -X POST http://localhost:4001/api/cloud/register \
  -H "Content-Type: application/json" \
  -d '{"name": "My Project", "email": "user@example.com"}'

# 3. Publish an event
curl -X POST http://localhost:4001/ws \
  -H "X-Tenant-Key: pk_..." \
  -H "X-User-ID: user-123" \
  ...
```

## Build

```bash
go build -o gateway-cloud ./cmd/server
./gateway-cloud
```

## Environment Variables

| Variable | Required | Default |
|---|---|---|
| DATABASE_URL | Yes | — |
| REDIS_URL | Yes | — |
| JWT_SECRET | Yes | — |
| ALLOWED_ORIGINS | Yes | — |
| PORT | No | 4001 |
| STRIPE_SECRET_KEY | No | — |
| STRIPE_WEBHOOK_SECRET | No | — |
