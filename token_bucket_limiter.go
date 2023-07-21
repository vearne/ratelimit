package ratelimit

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	slog "github.com/vearne/simplelog"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
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

	/*
		If the traffic is too large, the limiter will request Redis frequently.
		To avoid this situation, the frequency of accessing Redis will be limited.
	*/
	AntiDDoS        bool
	antiDDoSLimiter *rate.Limiter
	/*
		Get the token in advance before actually needing to use it
	*/
	EnablePreFetch bool
	PreFetchCount  int64
}

type Option func(*TokenBucketLimiter)

func NewTokenBucketRateLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int, maxCapacity int,
	batchSize int, opts ...Option) (Limiter, error) {

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
		AntiDDoS:         true,
		EnablePreFetch:   false, // default value
		PreFetchCount:    5,     // default value
	}
	r.interval = duration / time.Duration(throughput)
	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(&r)
	}

	if !r.redisClient.ScriptExists(ctx, r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(ctx, script).Val()
	}

	// 2x throughput
	r.antiDDoSLimiter = rate.NewLimiter(rate.Limit(r.throughputPerSec*2), maxCapacity*2)

	// Get the token before actually needing to use it
	if r.EnablePreFetch {
		go r.PreFetch()
	}
	return &r, nil
}

func WithAntiDDos(antiDDoS bool) Option {
	return func(r *TokenBucketLimiter) {
		r.AntiDDoS = antiDDoS
	}
}

func WithEnablePreFetch(preFetch bool) Option {
	return func(r *TokenBucketLimiter) {
		r.EnablePreFetch = preFetch
	}
}

func WithPreFetchCount(preFetchCount int64) Option {
	return func(r *TokenBucketLimiter) {
		r.PreFetchCount = preFetchCount
	}
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

// true: Actually get token in redis
// false: not need to get
func (r *TokenBucketLimiter) tryPreFetch() bool {
	if r.needFetch() {
		// try to get from redis
		// single flight
		_, err, _ := r.g.Do(r.key, func() (interface{}, error) {
			x, err := r.redisClient.EvalSha(
				context.Background(),
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
			r.N += x.(int64)
			r.Unlock()
			return r.N, nil
		})
		if err != nil {
			slog.Error("get token from redis:%v", err)
		}
		return true
	}
	return false
}

func (r *TokenBucketLimiter) needFetch() bool {
	r.Lock()
	defer r.Unlock()
	if r.N >= r.PreFetchCount {
		return false
	}
	return true
}

func (r *TokenBucketLimiter) PreFetch() {
	for {
		executeFlag := r.tryPreFetch()
		slog.Debug("tryPreFetch, %v", executeFlag)
		time.Sleep(10 * time.Millisecond)
	}
}

func (r *TokenBucketLimiter) Take(ctx context.Context) (bool, error) {
	// 0. Anti DDoS
	if r.AntiDDoS {
		if !r.antiDDoSLimiter.Allow() {
			return false, nil
		}
	}

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
		r.N += x.(int64)
		r.Unlock()
		return r.N, nil
	})

	if err != nil {
		return false, err
	}

	return r.tryTakeFromLocal(), nil
}
