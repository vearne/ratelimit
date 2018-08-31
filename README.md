# Distributed rate-limit library based on Redis

---

### Overview
The goal of this library is to be able to implement distributed rate limit functions simply and rudely. Similar to the usage of the ID generator, the client takes back the data from Redis - batch data (just a value here), as long as it Is not consumed.it doesn't exceed rate-limit.

* [中文 README](https://github.com/vearne/ratelimit/blob/master/README_zh.md)

### Advantage
* Less dependencies, only rely on Redis, no special services required
* use Redis own clock, The clients no need to have the same clock
* Thread (coroutine) security
* Low system overhead and little pressure on redis

### How to get
```
go get github.com/vearne/ratelimit
```
### Usage
#### 1. create redis.Client
with "github.com/go-redis/redis"
```
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xxx", // no password set
		DB:       0,  // use default DB
	})
```

#### 2. create RateLimiter
```
	limiter := ratelimit.NewRedisRateLimiter(client,
		"push",
		1 * time.Second,
		200,
		10,
	)
```
Indicates that 200 operations per second are allowed
```
	limiter := ratelimit.NewRedisRateLimiter(client,
		"push",
		1 * time.Minute,
		200,
		10,
	)
```
Indicates that 200 operations per minute are allowed

The function prototype is as follows:
```
func NewRedisRateLimiter(client *redis.Client, keyPrefix string,
	duration time.Duration, throughput int, batchSize int) (*RedisRateLimiter)
```
|参数|说明|
|:---|:---|
|keyPrefix|Key prefix in Redis|
|duration|Indicates that the operation `throughput` is allowed in the `duration` time interval|
|throughput|Indicates that the operation `throughput` is allowed in the `duration` time interval|
|batchSize|The number of available operations each time retrieved from redis|

**注意**
rate-limit = throughput / duration
In addition, in order to ensure that the performance is high enough, the minimum precision of duration is seconds.



#### Complete example
```
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
```


### Thanks
The development of the module was inspired by the Reference 1.



### Reference
1. [Performance million/s: Tencent lightweight global flow control program](http://wetest.qq.com/lab/view/320.html)




