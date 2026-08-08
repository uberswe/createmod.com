package ratelimit

import (
	"context"
	"time"
)

// Limiter is the interface for rate limiting operations.
// Implementations must be safe for concurrent use.
type Limiter interface {
	// Allow atomically increments the counter for key and returns whether
	// the request is within the limit. remaining is the number of requests
	// left in the current window (0 when denied).
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int)

	// Incr atomically increments the counter for key, setting window as its
	// TTL when the key is first created, and returns the post-increment count.
	// Returns 0 on backend error so callers treat a failure as "no signal".
	// Unlike Allow it exposes the raw count, which callers use to scale a
	// response (e.g. exponential backoff) to how far over a threshold a source is.
	Incr(ctx context.Context, key string, window time.Duration) int

	// Check returns true if key exists (used for dedup checks).
	Check(ctx context.Context, key string) bool

	// Mark sets key with the given TTL (used for dedup markers).
	Mark(ctx context.Context, key string, ttl time.Duration)

	// Close releases any resources held by the limiter.
	Close() error
}
