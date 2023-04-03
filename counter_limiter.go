package ratelimit

import (
	"context"
	"golang.org/x/sync/singleflight"
	"time"
)

//nolint:govet
type CounterLimiter struct {
	BaseRateLimiter
	duration   time.Duration
	throughput int
	batchSize  int
	N          int64
	g          singleflight.Group
}

func (r *CounterLimiter) tryTakeFromLocal() bool {
	r.Lock()
	defer r.Unlock()
	if r.N > 0 {
		r.N = r.N - 1
		return true
	}
	return false
}

func (r *CounterLimiter) Take(ctx context.Context) (bool, error) {
	// 1. try to get from local
	if r.tryTakeFromLocal() {
		return true, nil
	}

	// 2. try to get from redis
	_, err, _ := r.g.Do(r.key, func() (interface{}, error) {
		x, err := r.redisClient.EvalSha(
			ctx,
			r.scriptSHA1,
			[]string{r.key},
			int(r.duration/time.Microsecond),
			r.throughput,
			r.batchSize,
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
