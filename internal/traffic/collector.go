// Package traffic aggregates raw view/download hit counts in memory and flushes
// them to Postgres periodically, so bot-traffic analysis costs ~one batched
// write per interval instead of a row per request. It is observability only —
// it does not affect how views/downloads are counted for public stats.
package traffic

import (
	"context"
	"sync"
	"time"

	"createmod/internal/store"
)

const (
	flushInterval = 60 * time.Second
	maxUALen      = 512
)

type key struct {
	day        string
	eventType  string
	userAgent  string
	country    string
	resolution string
	pageClass  string
}

// Collector holds the in-memory deltas. The number of distinct keys is bounded
// by distinct (UA, country) pairs per day — not by request volume — so a flood
// grows counts, not the map.
type Collector struct {
	mu     sync.Mutex
	counts map[key]int64
	store  store.TrafficStatsStore
}

var def *Collector

// Init creates the process-wide collector and starts its flush loop. Call once
// at startup; a nil store disables collection. The loop stops (after a final
// drain) when ctx is cancelled.
func Init(ctx context.Context, s store.TrafficStatsStore) {
	if s == nil || def != nil {
		return
	}
	def = &Collector{counts: make(map[key]int64), store: s}
	go def.run(ctx)
}

// Record counts one raw hit. No-op until Init runs. eventType is "view",
// "view_js" or "download"; ua and country come from the request (country via
// CF-IPCountry). resolution is set only for the JS beacon ("view_js").
func Record(eventType, ua, country, resolution, pageClass string) {
	if def != nil {
		def.record(eventType, ua, country, resolution, pageClass)
	}
}

func (c *Collector) record(eventType, ua, country, resolution, pageClass string) {
	if len(ua) > maxUALen {
		ua = ua[:maxUALen]
	}
	k := key{
		day:        time.Now().UTC().Format("2006-01-02"),
		eventType:  eventType,
		userAgent:  ua,
		country:    country,
		resolution: resolution,
		pageClass:  pageClass,
	}
	c.mu.Lock()
	c.counts[k]++
	c.mu.Unlock()
}

func (c *Collector) run(ctx context.Context) {
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flush(context.Background()) // final drain
			return
		case <-t.C:
			c.flush(ctx)
		}
	}
}

func (c *Collector) flush(ctx context.Context) {
	c.mu.Lock()
	if len(c.counts) == 0 {
		c.mu.Unlock()
		return
	}
	snapshot := c.counts
	c.counts = make(map[key]int64)
	c.mu.Unlock()

	rows := make([]store.TrafficStatRow, 0, len(snapshot))
	for k, n := range snapshot {
		rows = append(rows, store.TrafficStatRow{
			Day:        k.day,
			EventType:  k.eventType,
			UserAgent:  k.userAgent,
			Country:    k.country,
			Resolution: k.resolution,
			PageClass:  k.pageClass,
			Count:      n,
		})
	}

	fctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := c.store.UpsertBatch(fctx, rows); err != nil {
		// Best-effort: re-merge the failed deltas so a transient DB error
		// doesn't lose counts; they flush again next tick. Keys are bounded, so
		// this can't grow without limit.
		c.mu.Lock()
		for k, n := range snapshot {
			c.counts[k] += n
		}
		c.mu.Unlock()
	}
}
