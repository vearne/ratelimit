package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit"
	"github.com/vearne/ratelimit/counter"
	"github.com/vearne/ratelimit/tokenbucket"
	slog "github.com/vearne/simplelog"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *counter.Counter, targetCount int) {
	defer group.Done()
	var ok bool
	for {
		ok = true
		err := r.Wait(context.Background())
		slog.Debug("r.Wait:%v", err)
		if err != nil {
			ok = false
			slog.Error("error:%v", err)
		}
		if ok {
			value := c.Incr()
			slog.Debug("---value--:%v", value)
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

	limiter, err := tokenbucket.NewTokenBucketRateLimiter(
		context.Background(),
		client,
		"key:token",
		time.Second,
		10,
		5,
		2)

	if err != nil {
		fmt.Println("error", err)
		return
	}

	var wg sync.WaitGroup
	total := 50
	counter := counter.NewCounter()
	start := time.Now()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
