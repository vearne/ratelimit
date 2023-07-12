package ratelimit

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	slog "github.com/vearne/simplelog"
	"golang.org/x/time/rate"
	"time"
)

type LeakyBucketLimiter struct {
	BaseRateLimiter

	// For interval between requests,the smallest unit of duration is one microseconds.
	interval time.Duration

	/*
		If the traffic is too large, the limiter will request Redis frequently.
		To avoid this situation, the frequency of accessing Redis will be limited.
	*/
	AntiDDoS        bool
	antiDDoSLimiter *rate.Limiter
}

func NewLeakyBucketLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int) (Limiter, error) {

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

	script := algMap[LeakyBucketAlg]
	scriptSHA1 := fmt.Sprintf("%x", sha1.Sum([]byte(script)))

	r := LeakyBucketLimiter{
		BaseRateLimiter: BaseRateLimiter{redisClient: client, scriptSHA1: scriptSHA1, key: key},
		interval:        duration / time.Duration(throughput),
		AntiDDoS:        true,
	}

	if !r.redisClient.ScriptExists(ctx, r.scriptSHA1).Val()[0] {
		r.redisClient.ScriptLoad(ctx, script).Val()
	}

	throughputPerSec := int(float64(throughput) / float64(duration/time.Second))
	r.antiDDoSLimiter = rate.NewLimiter(rate.Limit(throughputPerSec*2), throughputPerSec*2)

	return &r, nil
}

// just for test
func (r *LeakyBucketLimiter) WithAntiDDos(antiDDoS bool) {
	r.AntiDDoS = antiDDoS
}

// wait until take a token or timeout
func (r *LeakyBucketLimiter) Wait(ctx context.Context) (err error) {
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

func (r *LeakyBucketLimiter) Take(ctx context.Context) (bool, error) {
	// 0. Anti DDoS
	if r.AntiDDoS {
		if !r.antiDDoSLimiter.Allow() {
			return false, nil
		}
	}

	// 1. try to get from redis
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
