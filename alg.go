package ratelimit

const (
	TokenBucketAlg = iota
	CounterAlg

)

const counterScript = `
local key_prefix = KEYS[1]
local throughput_per_sec = tonumber(ARGV[1])
local batch_size = tonumber(ARGV[2])
local current_timestamp = redis.call("TIME")
local key = key_prefix .. ":" .. current_timestamp[1]
local n = redis.call("GET", key)
if n == false then
    n = 0
else
    n = tonumber(n)
end
if n >= throughput_per_sec then
    return 0
end
local increment = math.min(throughput_per_sec - n, batch_size)
redis.replicate_commands();
redis.call("INCRBY", key, increment)
redis.call("EXPIRE", key, 3)
return increment
`

/*
	key Type: Hash

	key ->
		token_count -> {token_count}
		updateTime -> {lastUpdateTime}   (microsecond)
 */

const TokenBucketScript = `
local bucket = KEYS[1]
local throughput_per_sec = tonumber(ARGV[1])
local batch_size = tonumber(ARGV[2])
local max_capacity = throughput_per_sec

local count = 0
local lastUpdateTime = redis.call("HGET", bucket, "updateTime")

if lastUpdateTime == false then
    lastUpdateTime = 0
end

local current_timestamp = tonumber(redis.call("TIME")[1]) * 1000000 + tonumber(redis.call("TIME")[2])
local increment = (current_timestamp - tonumber(lastUpdateTime)) / 1000000 * throughput_per_sec
local n = redis.call("HGET", bucket, "token_count")

increment = tonumber(increment)

if n == false then
    n = 0
else
    n = tonumber(n)
end

n = math.min(n + increment, max_capacity)

if n > batch_size then
	n = n - batch_size
	count = batch_size
else
	count = n
	n = 0
end

redis.replicate_commands();

redis.call("HSET", bucket, "token_count", n)
if increment >= 1 then
	redis.call("HSET", bucket, "updateTime", current_timestamp)
end

return count
`

var (
	algMap map[int]string
)


func init(){
	algMap = make(map[int]string)
	algMap[CounterAlg] = counterScript
	algMap[TokenBucketAlg] = TokenBucketScript
}