package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit"
	"math/rand"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, wg *sync.WaitGroup,
	c *ratelimit.Counter, targetCount int) {
	defer wg.Done()
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
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Millisecond)
		}
	}
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxeQl*@nFE", // password set
		DB:       0,            // use default DB
	})

	limiter, err := ratelimit.NewCounterRateLimiter(context.Background(), client,
		"key:count",
		time.Second,
		100,
		2,
	)

	if err != nil {
		fmt.Println("error", err)
		return
	}

	var wg sync.WaitGroup
	total := 500
	counter := ratelimit.NewCounter()
	start := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
