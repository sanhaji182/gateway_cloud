package cloud

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// go-redis/v9 adalah Redis client untuk Go — digunakan oleh core dan SaaS.
	"github.com/redis/go-redis/v9"
)

// PlanRateLimiter mengimplementasikan extensions.RateLimiter dari core.
//
// Algoritma: Redis token bucket sederhana per (tenant, action).
// Setiap Allow() akan INCR key Redis dengan TTL 1 menit. Jika counter
// melebihi batas plan, request ditolak dengan HTTP 429.
//
// Key pattern: rate:{tenantID}:{action} — contoh: rate:uuid-123:subscribe
//
// Filosofi "fail open": jika Redis error, Allow() return true agar
// request tidak diblokir hanya karena infrastruktur monitoring error.
type PlanRateLimiter struct {
	Redis *redis.Client // Redis client — shared dengan core Gateway.
	DB    *sql.DB       // PostgreSQL untuk query plan tier tenant.
}

// Allow memeriksa apakah request diizinkan berdasarkan rate limit tenant.
//
// Parameter:
//   - tenantID: UUID tenant (kosong = self-hosted, selalu allow)
//   - key:      nama action (ws_connect, subscribe, auth, publish)
//   - limit:    tidak dipakai — limit ditentukan oleh PlanLimits(plan)
//
// Return true jika masih dalam kuota, false jika perlu ditolak.
func (r PlanRateLimiter) Allow(tenantID, key string, _ int) bool {
	// Self-hosted user tidak punya tenant — selalu allow.
	if tenantID == "" {
		return true
	}
	// Query plan tenant untuk menentukan batas rate limit.
	var plan string
	err := r.DB.QueryRowContext(context.Background(),
		"SELECT plan FROM tenants WHERE id = $1", tenantID,
	).Scan(&plan)
	if err != nil {
		// Fail open: jika database error, jangan blokir traffic produksi.
		return true
	}
	limit, _ := PlanLimits(plan)
	// Redis INCR + EXPIRE: counter otomatis reset setiap menit.
	bucketKey := fmt.Sprintf("rate:%s:%s", tenantID, key)
	count, err := r.Redis.Incr(context.Background(), bucketKey).Result()
	if err != nil {
		return true // fail open: Redis error tidak memblokir request.
	}
	// Set TTL 1 menit hanya saat first INCR (count == 1).
	if count == 1 {
		r.Redis.Expire(context.Background(), bucketKey, time.Minute)
	}
	return count <= int64(limit)
}
