package ratelimit

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	slog "github.com/vearne/simplelog"
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

func NewTokenBucketRateLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int, maxCapacity int,
	batchSize int) (Limiter, error) {

	_, err := client.Ping(ctx).Result()
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
	r.interval = duration / time.Duration(throughput)

	if !r.redisClient.ScriptExists(ctx, r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(ctx, script).Val()
	}

	return &r, nil
}

// wait until take a token or timeout
func (r *TokenBucketLimiter) Wait(ctx context.Context) (err error) {
	ok, err := r.Take(ctx)
	slog.Debug("r.Take")
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	deadline, ok := ctx.Deadline()
	minWaitTime := r.interval
	slog.Debug("minWaitTime:%v", minWaitTime)
	if ok {
		if deadline.Before(time.Now().Add(minWaitTime)) {
			slog.Debug("can't get token before %v", deadline)
			return fmt.Errorf("can't get token before %v", deadline)
		}
	}

	for {
		slog.Debug("---for---")
		timer := time.NewTimer(minWaitTime)
		select {
		// 执行的代码
		case <-ctx.Done():
			return errors.New("context timeout")
		case <-timer.C:
			ok, err := r.Take(ctx)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
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
