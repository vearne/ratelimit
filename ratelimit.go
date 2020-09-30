package ratelimit

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

func NewCounterRateLimiter(client redis.Cmdable, key string, duration time.Duration,
	throughput int,
	batchSize int) (Limiter, error) {

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	if duration < time.Millisecond {
		return nil, errors.New("duration is too small")
	}

	if throughput <= 0 {
		return nil, errors.New("duration must greater than 0")
	}

	if batchSize <= 0 {
		return nil, errors.New("batchSize must greater than 0")
	}

	script := algMap[CounterAlg]
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(script)))

	r := CounterLimiter{
		BaseRateLimiter: BaseRateLimiter{redisClient: client, scriptSHA1: scriptSHA1, key: key},
		duration:        duration,
		throughput:      throughput,
		batchSize:       batchSize,
		N:               0,
	}

	if !r.redisClient.ScriptExists(r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(script).Val()
	}

	return &r, nil
}

func NewTokenBucketRateLimiter(client redis.Cmdable, key string, duration time.Duration,
	throughput int, maxCapacity int,
	batchSize int) (Limiter, error) {

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	if duration < time.Millisecond {
		return nil, errors.New("duration is too small")
	}

	if throughput <= 0 {
		return nil, errors.New("duration must greater than 0")
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

	if !r.redisClient.ScriptExists(r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(script).Val()
	}

	return &r, nil
}

func NewLeakyBucketLimiter(client redis.Cmdable, key string, duration time.Duration,
	throughput int) (Limiter, error) {

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	if duration < time.Millisecond {
		return nil, errors.New("duration is too small")
	}

	if throughput <= 0 {
		return nil, errors.New("duration must greater than 0")
	}

	script := algMap[LeakyBucketAlg]
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(script)))

	r := LeakyBucketLimiter{
		BaseRateLimiter: BaseRateLimiter{redisClient: client, scriptSHA1: scriptSHA1, key: key},
		interval:        duration / time.Duration(throughput),
	}

	if !r.redisClient.ScriptExists(r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(script).Val()
	}

	return &r, nil
}
