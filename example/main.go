package main

import (
	"github.com/go-redis/redis"
	"github.com/vearne/ratelimit"
	"time"
	"sync"
	"fmt"
	"math/rand"
)

func consume(r *ratelimit.RedisRateLimiter, group *sync.WaitGroup){
	for {
		if r.Take(){
			group.Done()
		}else{
			time.Sleep(time.Duration(rand.Intn(10) + 1)* time.Millisecond)
		}
	}
}


func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxx", // no password set
		DB:       0,  // use default DB
	})

	limiter := ratelimit.NewRedisRateLimiter(client,
		"push",
		1 * time.Second,
		200,
		10,
	)

	var wg sync.WaitGroup
	wg.Add(10000)
	start := time.Now()
	for i:=0;i<100;i++{
		go consume(limiter, &wg)
	}
	wg.Wait()
	fmt.Println("limit", time.Since(start))
}

