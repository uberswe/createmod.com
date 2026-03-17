package slowlog

import (
	"log/slog"
	"net/http"
	"time"
)

// SlowHTTPTransport wraps an http.RoundTripper and logs requests that exceed
// Threshold. Subsystem is included in the log entry so callers can distinguish
// between different HTTP clients (e.g. "openai", "modmeta", "moderation").
type SlowHTTPTransport struct {
	Base      http.RoundTripper
	Subsystem string
}

var _ http.RoundTripper = (*SlowHTTPTransport)(nil)

// RoundTrip executes the request via the base transport and logs a warning if
// the round-trip duration exceeds Threshold.
func (t *SlowHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	start := time.Now()
	resp, err := base.RoundTrip(req)
	elapsed := time.Since(start)

	if elapsed >= Threshold {
		var errVal any
		if err != nil {
			errVal = err.Error()
		}
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		slog.Warn("slow http request",
			"subsystem", t.Subsystem,
			"duration_ms", elapsed.Milliseconds(),
			"method", req.Method,
			"url", req.URL.String(),
			"status", status,
			"error", errVal,
		)
	}

	return resp, err
}
