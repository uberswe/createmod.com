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

	// Check returns true if key exists (used for dedup checks).
	Check(ctx context.Context, key string) bool

	// Mark sets key with the given TTL (used for dedup markers).
	Mark(ctx context.Context, key string, ttl time.Duration)

	// Close releases any resources held by the limiter.
	Close() error
}
