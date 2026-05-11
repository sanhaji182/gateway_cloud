package cloud

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PlanRateLimiter mengimplementasikan extensions.RateLimiter.
// Menggunakan Redis token bucket per tenant + plan tier.
type PlanRateLimiter struct {
	Redis *redis.Client
	DB    *sql.DB
}

func (r PlanRateLimiter) Allow(tenantID, key string, _ int) bool {
	if tenantID == "" {
		return true // self-hosted — always allow
	}
	// tenantID = UUID dari autentikasi — query plan langsung via ID.
	var plan string
	err := r.DB.QueryRowContext(context.Background(),
		"SELECT plan FROM tenants WHERE id = $1", tenantID,
	).Scan(&plan)
	if err != nil {
		return true // fail open
	}
	limit, _ := PlanLimits(plan)
	bucketKey := fmt.Sprintf("rate:%s:%s", tenantID, key)
	ctx := context.Background()
	count, err := r.Redis.Incr(ctx, bucketKey).Result()
	if err != nil {
		return true // fail open
	}
	if count == 1 {
		r.Redis.Expire(ctx, bucketKey, time.Minute)
	}
	return count <= int64(limit)
}
