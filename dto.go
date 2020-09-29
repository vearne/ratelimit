package ratelimit

import (
	"fmt"
	"github.com/go-redis/redis"
	"sync"
	"time"
)

type BaseRateLimiter struct {
	sync.Mutex
	redisClient redis.Cmdable
	scriptSHA1  string
	key         string
}

type CounterLimiter struct {
	BaseRateLimiter

	duration    time.Duration
	throughput  int
	batchSize   int
	N           int64
}


func (r *CounterLimiter) Take() bool{
	// 1. try to get from local
	r.Lock()

	if r.N > 0 {
		r.N = r.N - 1
		r.Unlock()
		return true
	}

	// 2. try to get from redis
	count := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.key},
		int(r.duration/ time.Microsecond),
		r.throughput,
		r.batchSize,
	).Val().(int64)

	if count <= 0 {
		r.Unlock()
		return false
	} else {
		r.N = count
		r.N--
		r.Unlock()
		return true
	}
}

type TokenBucketLimiter struct {
	BaseRateLimiter

	throughputPerSec float64
	batchSize   int
	maxCapacity int
	N           int64
}

func (r *TokenBucketLimiter) Take() bool{
	// 1. try to get from local
	r.Lock()

	if r.N > 0 {
		r.N = r.N - 1
		r.Unlock()
		return true
	}

	// 2. try to get from redis
	x, err := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.key},
		r.throughputPerSec,
		r.batchSize,
		r.maxCapacity,
	).Result()

	if err!= nil{
		fmt.Println(err)
	}
	count := x.(int64)

	if count <= 0 {
		r.Unlock()
		return false
	} else {
		r.N = count
		r.N--
		r.Unlock()
		return true
	}
}



type LeakyBucketLimiter struct {
	BaseRateLimiter

	// For interval between requests,the smallest unit of duration is one microseconds.
	interval time.Duration
}

func (r *LeakyBucketLimiter) Take() bool {
	// try to get from redis
	count := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.key},
		r.interval/time.Microsecond,
	).Val().(int64)

	if count <= 0 {
		return false
	} else {
		return true
	}
}
