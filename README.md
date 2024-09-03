# Distributed rate-limit library based on Redis
[![golang-ci](https://github.com/vearne/ratelimit/actions/workflows/golang-ci.yml/badge.svg)](https://github.com/vearne/ratelimit/actions/workflows/golang-ci.yml)

---

### Overview
The goal of this library is to be able to implement distributed rate limit functions simply and rudely. Similar to the usage of the ID generator, the client takes back the data from Redis - batch data (just a value here), as long as it Is not consumed.it doesn't exceed rate-limit.

* [中文 README](https://github.com/vearne/ratelimit/blob/master/README_zh.md)

### Advantage
* Less dependencies, only rely on Redis, no special services required
* use Redis own clock, The clients no need to have the same clock
* Thread (coroutine) security
* Low system overhead and little pressure on redis

## Notice
Different types of limiters may have different redis-key data types in redis.
So different types of limiters cannot use same name redis-key.

For example
```
127.0.0.1:6379> type key:leaky
string
127.0.0.1:6379> type key:token
hash
127.0.0.1:6379> hgetall key:token

"token_count"
"0"
"updateTime"
"1613805726567122"
127.0.0.1:6379> get key:leaky
"1613807035353864"
```

### How to get
```
go get github.com/vearne/ratelimit
```
### Usage
#### 1. create redis.Client
with "github.com/go-redis/redis"   
Supports both redis master-slave mode and cluster mode
```
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxx", // no password set
		DB:       0,  // use default DB
	})
```
```
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{"127.0.0.1:6379"},
		Password: "xxxx",
	})
```

#### 2. create RateLimiter
```
limiter, err := ratelimit.NewTokenBucketRateLimiter(ctx, client,                
        "push", time.Second, 200, 20, 5)
```
Indicates that 200 operations per second are allowed.
```
	limiter, err := ratelimit.NewTokenBucketRateLimiter(client, 
	        ctx, "push", time.Minute, 200, 20, 5)
```
Indicates that 200 operations per minute are allowed.


#### 2.1 Counter algorithm
```
func NewCounterRateLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int,
	batchSize int) (Limiter, error)
```

|parameter|Description|
|:---|:---|
|key|Key in Redis|
|duration|Indicates that the operation throughput is allowed in the duration time interval|
|throughput|Indicates that the operation throughput is allowed in the duration time interval|
|batchSize|The number of available operations each time retrieved from redis|

#### 2.2 Token bucket algorithm
```
func NewTokenBucketRateLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int, maxCapacity int,
	batchSize int) (Limiter, error)
```

|parameter|Description|
|:---|:---|
|key|Key in Redis|
|duration|Indicates that the operation throughput is allowed in the duration time interval|
|throughput|Indicates that the operation throughput is allowed in the duration time interval|
|maxCapacity|The maximum number of tokens that can be stored in the token bucket|
|batchSize|The number of available operations each time retrieved from redis|

#### 2.3 Leaky bucket algorithm
```
func NewLeakyBucketLimiter(ctx context.Context, client redis.Cmdable, key string, duration time.Duration,
	throughput int) (Limiter, error) 
```

|parameter|Description|
|:---|:---|
|key|Key in Redis|
|duration|Indicates that the operation throughput is allowed in the duration time interval|
|throughput|Indicates that the operation throughput is allowed in the duration time interval|

#### 2.4 sliding time window
```
NewSlideTimeWindowLimiter(throught int, duration time.Duration, windowBuckets int) (Limiter, error)
```

|parameter|Description|
|:---|:---|
|duration|Indicates that the operation throughput is allowed in the duration time interval|
|throughput|Indicates that the operation throughput is allowed in the duration time interval|
|windowBuckets|Indicates that windowBuckets buckets will be created for duration, and the time range represented by each bucket is duration/windowBuckets|

Note: This limiter is based on memory and does not rely on Redis, so it may not be used in distributed frequency limiting scenarios.

### example
[more example](https://github.com/vearne/ratelimit/tree/master/example)

```
package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/vearne/ratelimit"
	slog "github.com/vearne/simplelog"
	"sync"
	"time"
)

func consume(r ratelimit.Limiter, group *sync.WaitGroup,
	c *ratelimit.Counter, targetCount int) {
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

	limiter, err := ratelimit.NewTokenBucketRateLimiter(
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
	counter := ratelimit.NewCounter()
	start := time.Now()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go consume(limiter, &wg, counter, total)
	}
	wg.Wait()
	cost := time.Since(start)
	fmt.Println("cost", time.Since(start), "rate", float64(total)/cost.Seconds())
}
```

### Dependency
[go-redis/redis](https://github.com/go-redis/redis)

### Thanks
The development of the module was inspired by the Reference 1.



### Reference
1. [Performance million/s: Tencent lightweight global flow control program](http://wetest.qq.com/lab/view/320.html)


### Thanks
[![jetbrains](img/jetbrains.svg)](https://www.jetbrains.com/community/opensource/#support)


