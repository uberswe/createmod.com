// Package slowlog provides hooks and wrappers that log operations exceeding
// a configurable duration threshold. It supports PostgreSQL (via pgx tracer),
// Redis (via go-redis hook), S3 (via helper function), and arbitrary HTTP
// clients (via http.RoundTripper wrapper).
package slowlog

import "time"

// Threshold is the duration above which an operation is considered slow and
// logged at Warn level.
const Threshold = 500 * time.Millisecond
