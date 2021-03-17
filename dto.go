package ratelimit

import (
	"github.com/go-redis/redis"
	"sync"
	"time"
)

type Limiter interface {
	Take() (bool, error)
}

type BaseRateLimiter struct {
	scriptSHA1  string
	key         string
	redisClient redis.Cmdable
	sync.Mutex
}

type CounterLimiter struct {
	BaseRateLimiter
	duration   time.Duration
	throughput int
	batchSize  int
	N          int64
}

func (r *CounterLimiter) Take() (bool, error) {
	// 1. try to get from local
	r.Lock()

	if r.N > 0 {
		r.N = r.N - 1
		r.Unlock()
		return true, nil
	}

	// 2. try to get from redis
	x, err := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.key},
		int(r.duration/time.Microsecond),
		r.throughput,
		r.batchSize,
	).Result()

	if err != nil {
		return false, err
	}

	count := x.(int64)

	if count <= 0 {
		r.Unlock()
		return false, nil
	} else {
		r.N = count
		r.N--
		r.Unlock()
		return true, nil
	}
}

type TokenBucketLimiter struct {
	BaseRateLimiter

	throughputPerSec float64
	batchSize        int
	maxCapacity      int
	N                int64
}

func (r *TokenBucketLimiter) Take() (bool, error) {
	// 1. try to get from local
	r.Lock()

	if r.N > 0 {
		r.N = r.N - 1
		r.Unlock()
		return true, nil
	}

	// 2. try to get from redis
	x, err := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.key},
		r.throughputPerSec,
		r.batchSize,
		r.maxCapacity,
	).Result()

	if err != nil {
		return false, err
	}

	count := x.(int64)

	if count <= 0 {
		r.Unlock()
		return false, nil
	} else {
		r.N = count
		r.N--
		r.Unlock()
		return true, nil
	}
}

type LeakyBucketLimiter struct {
	BaseRateLimiter

	// For interval between requests,the smallest unit of duration is one microseconds.
	interval time.Duration
}

func (r *LeakyBucketLimiter) Take() (bool, error) {
	// try to get from redis
	x, err := r.redisClient.EvalSha(
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
