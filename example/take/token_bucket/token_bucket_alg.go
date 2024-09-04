package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit"
	"github.com/vearne/ratelimit/counter"
	"github.com/vearne/ratelimit/tokenbucket"
	"math/rand"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *counter.Counter, targetCount int) {
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
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Millisecond)
		}
	}
}

func main() {
	//slog.SetLevel(slog.DebugLevel)
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxeQl*@nFE", // password set
		DB:       0,            // use default DB
	})

	limiter, err := tokenbucket.NewTokenBucketRateLimiter(context.Background(), client, "key:token",
		time.Second,
		100,
		50,
		5,
		tokenbucket.WithEnablePreFetch(true),
		tokenbucket.WithPreFetchCount(10),
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
		wg.Add(1)
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
