package main

import (
	"context"
	"fmt"
	"github.com/vearne/ratelimit"
	"github.com/vearne/ratelimit/counter"
	"github.com/vearne/ratelimit/timewindow"
	"math/rand"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *counter.Counter, targetCount int) {
	group.Add(1)
	defer group.Done()
	for {
		ok, err := r.Take(context.Background())
		if err != nil {
			ok = true
			fmt.Println("error", err)
		}
		if ok {
			value := c.Incr()
			if value >= targetCount {
				break
			}
		} else {
			time.Sleep(time.Duration(rand.Intn(2)+1) * time.Millisecond)
		}
	}
}
func main() {
	limiter, _ := timewindow.NewSlideTimeWindowLimiter(100, time.Second, 10)

	var wg sync.WaitGroup
	total := 500
	counter := counter.NewCounter()
	start := time.Now()
	for i := 0; i < 100; i++ {
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
