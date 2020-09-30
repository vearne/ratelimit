package ratelimit

const (
	TokenBucketAlg = iota
	CounterAlg
	LeakyBucketAlg
)

const counterScript = `
local key_prefix = KEYS[1]
-- unit is microseconds
local unit = tonumber(ARGV[1])
local throughput = tonumber(ARGV[2])
local batch_size = tonumber(ARGV[3])
local timestamp = redis.call("TIME")
local current_timestamp = tonumber(timestamp[1]) * 1000000 + tonumber(timestamp[2])
local key = key_prefix .. ":" .. math.floor(current_timestamp/unit)
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
redis.call("EXPIRE", key, 3 * unit/1000000)
return increment
`

/*
	key Type: Hash

	key ->
		token_count -> {token_count}
		updateTime -> {lastUpdateTime}* 1000000  +  {microsecond}
 */

const TokenBucketScript = `
local bucket = KEYS[1]
local throughput_per_sec = tonumber(ARGV[1])
local batch_size = tonumber(ARGV[2])
local max_capacity = tonumber(ARGV[3])

local count = 0
local lastUpdateTime = redis.call("HGET", bucket, "updateTime")

if lastUpdateTime == false then
    lastUpdateTime = 0
end

local timestamp = redis.call("TIME")
local current_timestamp = tonumber(timestamp[1]) * 1000000 + tonumber(timestamp[2])
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


/*
	key Type:  string

    // updateTime
	key -> {lastUpdateTime}* 1000000  +  {microsecond}

 */
const LeakyBucketScript = `
local bucket = KEYS[1]
local interval = tonumber(ARGV[1])

local count = 0
local lastUpdateTime = redis.call("GET", bucket)

if lastUpdateTime == false then
    lastUpdateTime = 0
end

local current_timestamp = tonumber(redis.call("TIME")[1]) * 1000000 + tonumber(redis.call("TIME")[2])

if current_timestamp > tonumber(lastUpdateTime) + interval then
	count = 1
	redis.replicate_commands();
	redis.call("SET", bucket, current_timestamp)
else
	count = 0
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
	algMap[LeakyBucketAlg] = LeakyBucketScript
}