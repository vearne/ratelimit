package main

import (
	"context"
	"fmt"
	"github.com/vearne/ratelimit"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *ratelimit.Counter, targetCount int) {
	group.Add(1)
	defer group.Done()
	var ok bool
	for {
		ok = true
		err := r.Wait(context.Background())
		if err != nil {
			ok = false
			fmt.Println("error", err)
		}
		if ok {
			value := c.Incr()
			if value >= targetCount {
				break
			}
		}
	}
}
func main() {
	limiter, _ := ratelimit.NewSlideTimeWindowLimiter(100, time.Second, 100)

	var wg sync.WaitGroup
	total := 500
	counter := ratelimit.NewCounter()
	start := time.Now()
	for i := 0; i < 100; i++ {
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
