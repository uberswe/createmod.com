package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"createmod/internal/ratelimit"

	"github.com/go-chi/chi/v5"
)

// mount wraps a trivial 200 handler with the dump-backoff middleware on a
// /r/{id} route so requests can carry a distinct id and client IP.
func mountDumpRouter(soft int) chi.Router {
	rl := ratelimit.NewMemory()
	mw := dumpBackoffMiddleware(rl, "test", "id", soft, time.Minute, time.Hour)
	r := chi.NewRouter()
	r.With(mw).Get("/r/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}

func doGet(r chi.Router, id, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/r/"+id, nil)
	req.Header.Set("CF-Connecting-IP", ip)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// A single IP pulling DISTINCT ids past the threshold gets throttled, and the
// Retry-After doubles for each id beyond it.
func TestDumpBackoff_DistinctIDsEscalate(t *testing.T) {
	const soft = 3
	r := mountDumpRouter(soft)
	const ip = "203.0.113.7"

	for i := 0; i < soft; i++ {
		if rec := doGet(r, "id-"+strconv.Itoa(i), ip); rec.Code != http.StatusOK {
			t.Fatalf("request %d within threshold: code=%d, want 200", i, rec.Code)
		}
	}

	// The next distinct ids are the 1st, 2nd, 3rd past the threshold →
	// Retry-After of 1s, 2s, 4s.
	for step, want := range []int{1, 2, 4} {
		rec := doGet(r, "over-"+strconv.Itoa(step), ip)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("over-threshold request %d: code=%d, want 429", step, rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != strconv.Itoa(want) {
			t.Fatalf("over-threshold request %d: Retry-After=%q, want %q", step, got, strconv.Itoa(want))
		}
	}
}

// Re-fetching the SAME id is free no matter how often — a heavy user of one
// resource (e.g. an active editor session) is never throttled.
func TestDumpBackoff_SameIDNeverThrottled(t *testing.T) {
	r := mountDumpRouter(2)
	const ip = "203.0.113.8"
	for i := 0; i < 50; i++ {
		if rec := doGet(r, "the-only-id", ip); rec.Code != http.StatusOK {
			t.Fatalf("repeat request %d: code=%d, want 200", i, rec.Code)
		}
	}
}

// erroringLimiter mimics a Redis outage: Incr always returns 0 (the fail-open
// signal) and Check always misses, so the middleware must fall back to its
// per-pod in-memory limiter and still throttle.
type erroringLimiter struct{}

func (erroringLimiter) Allow(context.Context, string, int, time.Duration) (bool, int) {
	return true, 0
}
func (erroringLimiter) Incr(context.Context, string, time.Duration) int { return 0 }
func (erroringLimiter) Check(context.Context, string) bool              { return false }
func (erroringLimiter) Mark(context.Context, string, time.Duration)     {}
func (erroringLimiter) Close() error                                    { return nil }

// When the shared limiter errors, protection degrades to the per-pod fallback
// rather than disabling — distinct ids past the threshold still get 429.
func TestDumpBackoff_DegradesToFallbackOnBackendError(t *testing.T) {
	const soft = 3
	mw := dumpBackoffMiddleware(erroringLimiter{}, "test", "id", soft, time.Minute, time.Hour)
	r := chi.NewRouter()
	r.With(mw).Get("/r/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	const ip = "203.0.113.9"

	for i := 0; i < soft; i++ {
		if rec := doGet(r, "id-"+strconv.Itoa(i), ip); rec.Code != http.StatusOK {
			t.Fatalf("request %d within threshold: code=%d, want 200", i, rec.Code)
		}
	}
	if rec := doGet(r, "over", ip); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over-threshold via fallback: code=%d, want 429", rec.Code)
	}
}

// The counter is per client IP: one IP tripping the wall does not affect another.
func TestDumpBackoff_PerIP(t *testing.T) {
	const soft = 2
	r := mountDumpRouter(soft)

	// Exhaust the noisy IP.
	for i := 0; i < soft+2; i++ {
		doGet(r, "n-"+strconv.Itoa(i), "198.51.100.1")
	}
	// A different IP starts fresh.
	if rec := doGet(r, "fresh", "198.51.100.2"); rec.Code != http.StatusOK {
		t.Fatalf("second IP throttled: code=%d, want 200", rec.Code)
	}
}
