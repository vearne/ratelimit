package ratelimit

import (
	"sync"
	"time"
)

type SlideTimeWindowRatelimit struct {
	sync.Mutex
	buckets           []int
	lastUpdateTime    time.Time
	durationPerBucket time.Duration
	throught          int
	duration          time.Duration
	windowBuckets     int
}

func NewSlideTimeWindowRatelimit(throught int, duration time.Duration, windowBuckets int) *SlideTimeWindowRatelimit {
	s := SlideTimeWindowRatelimit{buckets: make([]int, windowBuckets)}
	s.throught = throught
	s.durationPerBucket = duration / time.Duration(windowBuckets)
	s.duration = duration
	s.lastUpdateTime = time.Now()
	s.windowBuckets = windowBuckets
	for i := 0; i < windowBuckets; i++ {
		s.buckets[i] = 0
	}
	return &s
}

func (s *SlideTimeWindowRatelimit) Take() (bool, error) {
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
	if s.throught-s.Count() > 0 {
		s.buckets[nowBucketIndex]++
		s.lastUpdateTime = time.Now()
		return true, nil
	} else {
		return false, nil
	}
}

func (s *SlideTimeWindowRatelimit) Count() int {
	total := 0
	for i := 0; i < s.windowBuckets; i++ {
		total += s.buckets[i]
	}
	return total
}
