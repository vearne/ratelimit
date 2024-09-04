package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit"
	"github.com/vearne/ratelimit/counter"
	"github.com/vearne/ratelimit/leakybucket"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *counter.Counter, targetCount int) {
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
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxeQl*@nFE", // password set
		DB:       0,            // use default DB
	})

	limiter, err := leakybucket.NewLeakyBucketLimiter(
		context.Background(),
		client,
		"key:leaky",
		1*time.Second,
		100,
	)

	if err != nil {
		fmt.Println("error", err)
		return
	}

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
