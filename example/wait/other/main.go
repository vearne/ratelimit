package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit/tokenbucket"
	slog "github.com/vearne/simplelog"
	"time"
)

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
	start := time.Now()
	total := 100
	for i := 0; i < total; i++ {
		err = limiter.Wait(context.Background())
		slog.Error("err:%v", err)
	}
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
