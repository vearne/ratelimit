package ratelimit

import (
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
)

type Limiter interface {
	Take(ctx context.Context) (bool, error)
}

// nolint: govet
type BaseRateLimiter struct {
	sync.Mutex
	scriptSHA1  string
	key         string
	redisClient redis.Cmdable
}
