package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"math/rand"
	"ratelimit"
	"sync"
	"time"
)

func consume(r *ratelimit.RedisRateLimiter, group *sync.WaitGroup) {
	for {
		if r.Take() {
			group.Done()
		} else {
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Millisecond)
		}
	}
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxxxx", // password set
		DB:       0,       // use default DB
	})

	limiter, _ := ratelimit.NewRedisRateLimiter(client,
		"push",
		1*time.Second,
		100,
		10,
		//ratelimit.CounterAlg,
		ratelimit.TokenBucketAlg,
	)

	var wg sync.WaitGroup
	total := 5000
	wg.Add(total)
	start := time.Now()
	for i := 0; i < 100; i++ {
		go consume(limiter, &wg)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
