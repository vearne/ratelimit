package ratelimit

import (
	"context"
	"github.com/redis/go-redis/v9"
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
	ScriptSHA1  string
	Key         string
	RedisClient redis.Cmdable
	// For interval between requests,the smallest unit of duration is one microseconds.
	Interval time.Duration
}
