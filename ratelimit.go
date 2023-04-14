package ratelimit

import (
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

type Limiter interface {
	Take(ctx context.Context) (bool, error)
	Wait(ctx context.Context) (err error)
}

// nolint: govet
type BaseRateLimiter struct {
	sync.Mutex
	scriptSHA1  string
	key         string
	redisClient redis.Cmdable
	// For interval between requests,the smallest unit of duration is one microseconds.
	interval time.Duration
}
