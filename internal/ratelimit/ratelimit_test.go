package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiter_Allow(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test:allow"

	// First 3 requests should be allowed with limit=3
	for i := 0; i < 3; i++ {
		ok, remaining := m.Allow(ctx, key, 3, time.Minute)
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
		expected := 3 - (i + 1)
		if remaining != expected {
			t.Fatalf("request %d: expected remaining=%d, got %d", i+1, expected, remaining)
		}
	}

	// 4th request should be denied
	ok, remaining := m.Allow(ctx, key, 3, time.Minute)
	if ok {
		t.Fatal("4th request should be denied")
	}
	if remaining != 0 {
		t.Fatalf("expected remaining=0, got %d", remaining)
	}
}

func TestMemoryLimiter_AllowWindowExpiry(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test:expiry"

	// Use up the limit
	for i := 0; i < 2; i++ {
		m.Allow(ctx, key, 2, 50*time.Millisecond)
	}

	ok, _ := m.Allow(ctx, key, 2, 50*time.Millisecond)
	if ok {
		t.Fatal("should be denied after limit")
	}

	// Wait for the window to expire
	time.Sleep(60 * time.Millisecond)

	ok, remaining := m.Allow(ctx, key, 2, 50*time.Millisecond)
	if !ok {
		t.Fatal("should be allowed after window expires")
	}
	if remaining != 1 {
		t.Fatalf("expected remaining=1, got %d", remaining)
	}
}

func TestMemoryLimiter_CheckAndMark(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test:dedup"

	// Key should not exist
	if m.Check(ctx, key) {
		t.Fatal("key should not exist yet")
	}

	// Mark the key
	m.Mark(ctx, key, 100*time.Millisecond)

	// Key should exist
	if !m.Check(ctx, key) {
		t.Fatal("key should exist after Mark")
	}

	// Wait for TTL to expire
	time.Sleep(110 * time.Millisecond)

	// Key should be expired
	if m.Check(ctx, key) {
		t.Fatal("key should be expired")
	}
}

func TestMemoryLimiter_Close(t *testing.T) {
	m := NewMemory()
	if err := m.Close(); err != nil {
		t.Fatalf("Close should not error: %v", err)
	}
}
