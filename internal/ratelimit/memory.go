package ratelimit

import (
	"context"
	"sync"
	"time"
)

type memEntry struct {
	count  int
	expiry time.Time
}

// MemoryLimiter implements Limiter using an in-memory map with a background
// cleanup goroutine. Suitable for single-pod deployments or local development.
type MemoryLimiter struct {
	mu      sync.Mutex
	entries map[string]*memEntry
	stopCh  chan struct{}
}

// NewMemory creates a new in-memory rate limiter with a background cleanup
// goroutine that evicts expired entries every 60 seconds.
func NewMemory() *MemoryLimiter {
	m := &MemoryLimiter{
		entries: make(map[string]*memEntry),
		stopCh:  make(chan struct{}),
	}
	go m.cleanup()
	return m
}

func (m *MemoryLimiter) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for k, e := range m.entries {
				if now.After(e.expiry) {
					delete(m.entries, k)
				}
			}
			m.mu.Unlock()
		case <-m.stopCh:
			return
		}
	}
}

func (m *MemoryLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, ok := m.entries[key]
	if !ok || now.After(e.expiry) {
		m.entries[key] = &memEntry{count: 1, expiry: now.Add(window)}
		return true, limit - 1
	}
	e.count++
	remaining := limit - e.count
	if remaining < 0 {
		remaining = 0
	}
	return e.count <= limit, remaining
}

func (m *MemoryLimiter) Check(_ context.Context, key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.entries[key]
	if !ok {
		return false
	}
	return time.Now().Before(e.expiry)
}

func (m *MemoryLimiter) Mark(_ context.Context, key string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[key] = &memEntry{count: 1, expiry: time.Now().Add(ttl)}
}

func (m *MemoryLimiter) Close() error {
	close(m.stopCh)
	return nil
}
