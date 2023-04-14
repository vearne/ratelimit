package ratelimit

import (
	"context"
	"errors"
	"fmt"
	slog "github.com/vearne/simplelog"
	"sync"
	"time"
)

// nolint: govet
type SlideTimeWindowLimiter struct {
	sync.Mutex

	BaseRateLimiter

	duration      time.Duration
	throughput    int
	windowBuckets int

	durationPerBucket time.Duration
	lastUpdateTime    time.Time
	buckets           []int
}

func NewSlideTimeWindowLimiter(throughput int, duration time.Duration, windowBuckets int) (Limiter, error) {
	s := SlideTimeWindowLimiter{buckets: make([]int, windowBuckets)}
	s.throughput = throughput
	s.durationPerBucket = duration / time.Duration(windowBuckets)
	s.duration = duration
	s.lastUpdateTime = time.Now()
	s.windowBuckets = windowBuckets
	s.interval = duration / time.Duration(throughput)
	for i := 0; i < windowBuckets; i++ {
		s.buckets[i] = 0
	}
	return &s, nil
}

// wait until take a token or timeout
func (r *SlideTimeWindowLimiter) Wait(ctx context.Context) (err error) {
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
	fmt.Println("----1-----")
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
	return nil
}

func (s *SlideTimeWindowLimiter) Take(ctx context.Context) (bool, error) {
	s.Lock()
	defer s.Unlock()

	nowTime := time.Now()
	lastBucketIndex := int(s.lastUpdateTime.UnixNano()/int64(s.durationPerBucket)) % s.windowBuckets
	nowBucketIndex := int(nowTime.UnixNano()/int64(s.durationPerBucket)) % s.windowBuckets

	if nowTime.Sub(s.lastUpdateTime) > s.durationPerBucket*time.Duration(s.windowBuckets-1) {
		for i := 0; i < s.windowBuckets; i++ {
			s.buckets[i] = 0
		}
	} else if nowBucketIndex != lastBucketIndex {
		for i := (lastBucketIndex + 1) % s.windowBuckets; i != nowBucketIndex; i = (i + 1) % s.windowBuckets {
			s.buckets[i] = 0
		}
	}
	if s.throughput-s.Count() > 0 {
		s.buckets[nowBucketIndex]++
		s.lastUpdateTime = time.Now()
		return true, nil
	} else {
		return false, nil
	}
}

func (s *SlideTimeWindowLimiter) Count() int {
	total := 0
	for i := 0; i < s.windowBuckets; i++ {
		total += s.buckets[i]
	}
	return total
}
