package ratelimit

import (
	"github.com/go-redis/redis"
	"fmt"
	"crypto/sha1"
	"time"
	"sync"
)

// time_unit is the second of unit
// 最小精度是秒
const SCRIPT = `
local current_timestamp = redis.call("TIME")
local key_prefix = KEYS[1]
local duration_secs = tonumber(ARGV[1])
local throughput = tonumber(ARGV[2])
local batch_size = tonumber(ARGV[3])
local key = key_prefix .. ":" .. tostring(math.ceil(tonumber(current_timestamp[1])/duration_secs))
local n = redis.call("GET", key)

if n == false then
    n = 0
else
    n = tonumber(n)
end

if n >= throughput then
    return 0
end

local increment = math.min(throughput - n, batch_size)
redis.replicate_commands();
redis.call("INCRBY", key, increment)
redis.call("EXPIRE", key, duration_secs * 3)
return increment
`

type RedisRateLimiter struct {
	sync.Mutex
	redisClient  *redis.Client
	scriptSHA1   string
	keyPrefix          string
	durationSecs int
	throughput   int
	batchSize    int
	N            int64
}

// duration 精度最小到秒
//

func NewRedisRateLimiter(client *redis.Client, keyPrefix string,
	duration time.Duration, throughput int, batchSize int) (*RedisRateLimiter) {

	durationSecs := duration / time.Second
	if durationSecs < 1 {
		durationSecs = 1
	}

	r := &RedisRateLimiter{
		redisClient:  client,
		scriptSHA1:   fmt.Sprintf("%x", sha1.Sum([]byte(SCRIPT))),
		keyPrefix:          keyPrefix,
		durationSecs: int(durationSecs),
		throughput:   throughput,
		batchSize:    batchSize,
		N:            0,
	}

	if !r.redisClient.ScriptExists(r.scriptSHA1).Val()[0] {
		r.scriptSHA1 = r.redisClient.ScriptLoad(SCRIPT).Val()
	}
	return r
}

func (r *RedisRateLimiter) Take() bool {
	// 1. 尝试从本地获取
	r.Lock()

	if r.N > 0{
		r.N = r.N -1
		r.Unlock()
		return true
	}

	// 尝试从Redis获取
	count := r.redisClient.EvalSha(
		r.scriptSHA1,
		[]string{r.keyPrefix},
		r.durationSecs,
		r.throughput,
		r.batchSize,
	).Val().(int64)


	if count <= 0{
		r.Unlock()
		return false
	}else{
		r.N = count
		r.N--
		r.Unlock()
		return true
	}
}
