package traffic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"createmod/internal/store"
)

type fakeStore struct {
	rows     []store.TrafficStatRow
	failNext bool
}

func (f *fakeStore) UpsertBatch(_ context.Context, rows []store.TrafficStatRow) error {
	if f.failNext {
		f.failNext = false
		return errors.New("boom")
	}
	f.rows = append(f.rows, rows...)
	return nil
}
func (f *fakeStore) DeleteBefore(context.Context, string) error { return nil }

func newTestCollector(s store.TrafficStatsStore) *Collector {
	return &Collector{counts: make(map[key]int64), store: s}
}

// The collector must aggregate identical (type, UA, country) hits into one row
// with the summed count, and flush distinct buckets separately.
func TestCollector_AggregatesAndFlushes(t *testing.T) {
	fs := &fakeStore{}
	c := newTestCollector(fs)

	c.record("view", "botUA", "SG", "", "schematic")
	c.record("view", "botUA", "SG", "", "schematic")
	c.record("view", "botUA", "SG", "", "schematic")
	c.record("download", "botUA", "SG", "", "schematic")
	c.record("view", "realBrowser", "US", "", "schematic")

	c.flush(context.Background())

	byKey := map[string]int64{}
	for _, r := range fs.rows {
		byKey[r.EventType+"|"+r.UserAgent+"|"+r.Country] = r.Count
	}
	if got := byKey["view|botUA|SG"]; got != 3 {
		t.Errorf("view|botUA|SG = %d, want 3", got)
	}
	if got := byKey["download|botUA|SG"]; got != 1 {
		t.Errorf("download|botUA|SG = %d, want 1", got)
	}
	if got := byKey["view|realBrowser|US"]; got != 1 {
		t.Errorf("view|realBrowser|US = %d, want 1", got)
	}
	if len(fs.rows) != 3 {
		t.Errorf("flushed %d rows, want 3 distinct buckets", len(fs.rows))
	}

	// After a successful flush the buffer is empty — a second flush is a no-op.
	fs.rows = nil
	c.flush(context.Background())
	if len(fs.rows) != 0 {
		t.Errorf("second flush wrote %d rows, want 0", len(fs.rows))
	}
}

// A failed flush must re-merge deltas so nothing is lost; the next flush retries.
func TestCollector_ReMergesOnFailure(t *testing.T) {
	fs := &fakeStore{failNext: true}
	c := newTestCollector(fs)
	c.record("download", "ua", "US", "", "schematic")
	c.record("download", "ua", "US", "", "schematic")

	c.flush(context.Background()) // fails, re-merges
	if len(fs.rows) != 0 {
		t.Fatalf("failed flush should not have persisted rows, got %d", len(fs.rows))
	}
	c.flush(context.Background()) // retry succeeds
	if len(fs.rows) != 1 || fs.rows[0].Count != 2 {
		t.Errorf("retry rows = %+v, want one row count=2", fs.rows)
	}
}

// Over-long user agents are truncated so they can't blow past the index limit.
func TestCollector_TruncatesUA(t *testing.T) {
	fs := &fakeStore{}
	c := newTestCollector(fs)
	c.record("view", strings.Repeat("x", maxUALen+50), "US", "", "schematic")
	c.flush(context.Background())
	if len(fs.rows) != 1 || len(fs.rows[0].UserAgent) != maxUALen {
		t.Errorf("UA len = %d, want %d", len(fs.rows[0].UserAgent), maxUALen)
	}
}
