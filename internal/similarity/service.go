// Package similarity holds the per-pod in-memory fingerprint index for
// structure-similarity search. At current library scale (~5k schematics,
// ~5 KB per fingerprint) a brute-force scan is single-digit milliseconds;
// ANN indexing is deliberately absent until the library grows ~20x.
package similarity

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"createmod/internal/schematic"
	"createmod/internal/store"
)

type entry struct {
	id string
	fp *schematic.Fingerprint
}

// Service is the in-memory fingerprint index. Loaded from the store on boot
// and refreshed periodically (fingerprints are computed by River jobs on
// another pod; each pod pulls the shared table).
type Service struct {
	mu       sync.RWMutex
	entries  []entry
	appStore *store.Store
}

func New(appStore *store.Store) *Service {
	return &Service{appStore: appStore}
}

// Start loads the index and refreshes it every 10 minutes until ctx ends.
// While the index is empty (fresh deploy, backfill still running) it
// retries every 30 seconds so search comes online with the first batch.
func (s *Service) Start(ctx context.Context) {
	s.Reload(ctx)
	go func() {
		for {
			interval := 10 * time.Minute
			if s.Size() == 0 {
				interval = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				s.Reload(ctx)
			}
		}
	}()
}

// Reload replaces the index with the current fingerprint table contents.
func (s *Service) Reload(ctx context.Context) {
	if s.appStore == nil {
		return
	}
	rows, err := s.appStore.Fingerprints.ListAll(ctx, schematic.FingerprintVersion)
	if err != nil {
		slog.Warn("similarity: reload failed", "error", err)
		return
	}
	entries := make([]entry, 0, len(rows))
	for _, r := range rows {
		if len(r.FP) <= 2 { // "{}" placeholder for unparseable files
			continue
		}
		fp, err := schematic.DecodeFingerprint(r.FP)
		if err != nil || fp.Version != schematic.FingerprintVersion {
			continue
		}
		entries = append(entries, entry{id: r.SchematicID, fp: fp})
	}
	s.mu.Lock()
	s.entries = entries
	s.mu.Unlock()
	slog.Info("similarity: index loaded", "fingerprints", len(entries))
}

// Size returns the number of indexed fingerprints.
func (s *Service) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// Get returns the indexed fingerprint for a schematic, if present.
func (s *Service) Get(schematicID string) *schematic.Fingerprint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.id == schematicID {
			return e.fp
		}
	}
	return nil
}

// Result is one similar schematic with its similarity breakdown.
type Result struct {
	SchematicID string
	Similarity  schematic.Similarity
}

// FindSimilar scans the index and returns the top matches above minOverall,
// most similar first. excludeID removes the query schematic itself.
func (s *Service) FindSimilar(fp *schematic.Fingerprint, excludeID string, limit int, minOverall float64) []Result {
	if fp == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]Result, 0, 32)
	for _, e := range s.entries {
		if e.id == excludeID {
			continue
		}
		sim := schematic.Compare(fp, e.fp)
		if sim.Overall < minOverall {
			continue
		}
		results = append(results, Result{SchematicID: e.id, Similarity: sim})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Similarity.Overall > results[j].Similarity.Overall })
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}
