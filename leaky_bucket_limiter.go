package ratelimit

import (
	"context"
	"time"
)

type LeakyBucketLimiter struct {
	BaseRateLimiter

	// For interval between requests,the smallest unit of duration is one microseconds.
	interval time.Duration
}

func (r *LeakyBucketLimiter) Take(ctx context.Context) (bool, error) {
	// try to get from redis
	x, err := r.redisClient.EvalSha(
		ctx,
		r.scriptSHA1,
		[]string{r.key},
		int(r.interval/time.Microsecond),
	).Result()

	if err != nil {
		return false, err
	}

	count := x.(int64)

	if count <= 0 {
		return false, nil
	} else {
		return true, nil
	}
}
