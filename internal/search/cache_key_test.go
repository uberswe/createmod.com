package search

import (
	"testing"
	"time"
)

// The cache key must be scoped per environment: dev and prod share one S3
// bucket, and pods reload this cache every 10 minutes, so a shared key lets
// one environment's snapshot poison the other's in-memory index.
func Test_CacheKeyEnvScope(t *testing.T) {
	at := time.Date(2026, 7, 10, 15, 0, 0, 0, time.UTC)

	t.Setenv("ENVIRONMENT", "dev")
	if got := cacheKeyForDate(at); got != "_internal/search_index_cache_dev_2026-07-10.json.gz" {
		t.Errorf("dev key = %q", got)
	}

	t.Setenv("ENVIRONMENT", "production")
	if got := cacheKeyForDate(at); got != "_internal/search_index_cache_production_2026-07-10.json.gz" {
		t.Errorf("production key = %q", got)
	}

	t.Setenv("ENVIRONMENT", "")
	if got := cacheKeyForDate(at); got != "_internal/search_index_cache_local_2026-07-10.json.gz" {
		t.Errorf("local key = %q", got)
	}

	if got := unscopedCacheKeyForDate(at); got != "_internal/search_index_cache_2026-07-10.json.gz" {
		t.Errorf("unscoped key = %q", got)
	}
}
