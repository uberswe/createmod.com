package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// allowScript is a Lua script that atomically increments a counter and sets
// an expiry on first creation. This avoids the race condition where INCR
// succeeds but EXPIRE fails, leaving a key with no TTL.
//
// KEYS[1] = rate limit key
// ARGV[1] = window duration in seconds
// Returns: current count after increment
var allowScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

// RedisLimiter implements Limiter using Redis as the backing store.
type RedisLimiter struct {
	client *redis.Client
}

// NewRedis creates a new RedisLimiter by parsing the given Redis URL
// (e.g. "redis://localhost:6379/0") and verifying connectivity.
func NewRedis(redisURL string) (*RedisLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return &RedisLimiter{client: client}, nil
}

func (r *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int) {
	windowSec := int(window.Seconds())
	if windowSec < 1 {
		windowSec = 1
	}
	result, err := allowScript.Run(ctx, r.client, []string{key}, windowSec).Int()
	if err != nil {
		// On Redis error, fail open (allow the request).
		return true, limit
	}
	remaining := limit - result
	if remaining < 0 {
		remaining = 0
	}
	return result <= limit, remaining
}

func (r *RedisLimiter) Check(ctx context.Context, key string) bool {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return n > 0
}

func (r *RedisLimiter) Mark(ctx context.Context, key string, ttl time.Duration) {
	r.client.Set(ctx, key, "1", ttl)
}

func (r *RedisLimiter) Close() error {
	return r.client.Close()
}
