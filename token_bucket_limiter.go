package ratelimit

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/singleflight"
	"time"
)

//nolint:govet
type TokenBucketLimiter struct {
	BaseRateLimiter

	throughputPerSec float64
	batchSize        int
	maxCapacity      int
	N                int64

	g singleflight.Group
}

func NewTokenBucketRateLimiter(client redis.Cmdable, key string, duration time.Duration,
	throughput int, maxCapacity int,
	batchSize int) (Limiter, error) {

	bgCtx := context.Background()
	_, err := client.Ping(bgCtx).Result()
	if err != nil {
		return nil, err
	}

	if duration < time.Millisecond {
		return nil, errors.New("duration is too small")
	}

	if throughput <= 0 {
		return nil, errors.New("throughput must greater than 0")
	}

	if batchSize <= 0 {
		return nil, errors.New("batchSize must greater than 0")
	}

	script := algMap[TokenBucketAlg]
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(script)))

	r := TokenBucketLimiter{
		BaseRateLimiter:  BaseRateLimiter{redisClient: client, scriptSHA1: scriptSHA1, key: key},
		throughputPerSec: float64(throughput) / float64(duration/time.Second),
		maxCapacity:      maxCapacity,
		batchSize:        batchSize,
		N:                0,
	}

	if !r.redisClient.ScriptExists(bgCtx, r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(bgCtx, script).Val()
	}

	return &r, nil
}

func (r *TokenBucketLimiter) tryTakeFromLocal() bool {
	r.Lock()
	defer r.Unlock()
	if r.N > 0 {
		r.N = r.N - 1
		return true
	}
	return false
}

func (r *TokenBucketLimiter) Take(ctx context.Context) (bool, error) {
	// 1. try to get from local
	if r.tryTakeFromLocal() {
		return true, nil
	}

	// 2. try to get from redis
	// single flight
	_, err, _ := r.g.Do(r.key, func() (interface{}, error) {
		x, err := r.redisClient.EvalSha(
			ctx,
			r.scriptSHA1,
			[]string{r.key},
			r.throughputPerSec,
			r.batchSize,
			r.maxCapacity,
		).Result()
		if err != nil {
			return 0, err
		}
		r.Lock()
		r.N = x.(int64)
		r.Unlock()
		return r.N, nil
	})

	if err != nil {
		return false, err
	}

	return r.tryTakeFromLocal(), nil
}
