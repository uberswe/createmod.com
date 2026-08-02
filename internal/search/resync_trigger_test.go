package search

import (
	"context"
	"testing"
	"time"
)

func TestTriggerResyncIfEmpty(t *testing.T) {
	svc := &Service{index: []schematicIndex{{ID: "a"}}}
	m := &MeiliEngine{svc: svc}
	// Simulate a genuinely-empty Meilisearch index so the doc-count guard
	// allows the rebuild for the broad 0-result cases below.
	m.indexDocCount = func(ctx context.Context) (int64, error) { return 0, nil }

	// No enqueuer wired: must be a silent no-op.
	m.triggerResyncIfEmpty(SearchQuery{}, 0)

	var calls int
	m.SetResyncEnqueuer(func() { calls++ })

	// Non-broad queries and non-empty results never trigger.
	m.triggerResyncIfEmpty(SearchQuery{Term: "windmill"}, 0)
	m.triggerResyncIfEmpty(SearchQuery{Category: "automation"}, 0)
	m.triggerResyncIfEmpty(SearchQuery{}, 3)
	if calls != 0 {
		t.Fatalf("expected no enqueues yet, got %d", calls)
	}

	// A broad 0-result query against a confirmed-empty index triggers once...
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 1 {
		t.Fatalf("expected 1 enqueue, got %d", calls)
	}

	// ...and the per-pod cooldown swallows immediate repeats.
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 1 {
		t.Fatalf("cooldown violated: expected 1 enqueue, got %d", calls)
	}

	// Once the cooldown elapses it may trigger again.
	m.lastResync = time.Now().Add(-resyncTriggerCooldown - time.Second)
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 2 {
		t.Fatalf("expected 2 enqueues after cooldown, got %d", calls)
	}

	// An empty in-memory index means there is nothing to resync from.
	prevSvc := m.svc
	m.svc = &Service{}
	m.lastResync = time.Time{}
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 2 {
		t.Fatalf("expected no enqueue with empty in-memory index, got %d", calls)
	}
	m.svc = prevSvc
}

// The 2026-08-02 regression: a broad query returned 0 hits while Meilisearch
// actually held the full corpus (transient, under rebuild load). The guard must
// NOT rebuild in that case, or rebuilds feed back into more 0-results.
func TestTriggerResync_SkipsWhenIndexHasDocuments(t *testing.T) {
	svc := &Service{index: []schematicIndex{{ID: "a"}}}
	m := &MeiliEngine{svc: svc}
	m.indexDocCount = func(ctx context.Context) (int64, error) { return 7506, nil }

	var calls int
	m.SetResyncEnqueuer(func() { calls++ })

	// Broad 0-result, past cooldown, in-memory index populated — but the live
	// index reports 7506 docs, so no rebuild must be enqueued.
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 0 {
		t.Fatalf("rebuild storm: enqueued a rebuild while the index had documents (got %d)", calls)
	}

	// A stats-check error is also fail-safe: no rebuild when the count is unknown.
	m.lastResync = time.Time{}
	m.indexDocCount = func(ctx context.Context) (int64, error) { return 0, context.DeadlineExceeded }
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 0 {
		t.Fatalf("expected no rebuild when doc count is unknown, got %d", calls)
	}
}
