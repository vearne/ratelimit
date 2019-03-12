package ratelimit

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"sync"
	"time"
)

type RedisRateLimiter struct {
	sync.Mutex
	redisClient      *redis.Client
	scriptSHA1       string
	key              string
	throughputPerSec int
	batchSize        int
	N                int64
}

// the smallest unit of duration is one seconds
//

func NewRedisRateLimiter(client *redis.Client, key string,
	duration time.Duration, throughput int, batchSize int, alg int) (*RedisRateLimiter, error) {

	script, ok := algMap[alg]
	if !ok {
		return nil, errors.New("alg not support")
	}

	durationSecs := duration / time.Second
	if durationSecs < 1 {
		durationSecs = 1
	}

	r := &RedisRateLimiter{
		redisClient:      client,
		scriptSHA1:       fmt.Sprintf("%x", sha1.Sum([]byte(script))),
		key:              key,
		throughputPerSec: throughput / int(durationSecs),
		batchSize:        batchSize,
		N:                0,
	}

	if !r.redisClient.ScriptExists(r.scriptSHA1).Val()[0] {
		r.scriptSHA1 = r.redisClient.ScriptLoad(script).Val()
	}
	return r, nil
}

func (r *RedisRateLimiter) Take() bool {
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
		r.throughputPerSec,
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
