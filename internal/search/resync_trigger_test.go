package search

import (
	"testing"
	"time"
)

func TestTriggerResyncIfEmpty(t *testing.T) {
	svc := &Service{index: []schematicIndex{{ID: "a"}}}
	m := &MeiliEngine{svc: svc}

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

	// A broad 0-result query triggers exactly once...
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
	m.svc = &Service{}
	m.lastResync = time.Time{}
	m.triggerResyncIfEmpty(SearchQuery{}, 0)
	if calls != 2 {
		t.Fatalf("expected no enqueue with empty index, got %d", calls)
	}
}
