package jobs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/riverqueue/river"
)

// ConvCacheCleanupArgs sweeps the _conv/ format-conversion cache in S3,
// deleting objects that have not been touched in 30 days. Cache keys embed
// the schematic's Updated timestamp, so edited schematics leave stale keys
// behind; this job is what eventually reclaims them.
type ConvCacheCleanupArgs struct{}

func (ConvCacheCleanupArgs) Kind() string { return "conv_cache_cleanup" }

type ConvCacheCleanupWorker struct {
	river.WorkerDefaults[ConvCacheCleanupArgs]
	deps Deps
}

const convCacheTTL = 30 * 24 * time.Hour

func (w *ConvCacheCleanupWorker) Work(ctx context.Context, job *river.Job[ConvCacheCleanupArgs]) error {
	if w.deps.Storage == nil {
		return nil
	}
	cutoff := time.Now().Add(-convCacheTTL)
	var stale []string
	err := w.deps.Storage.ListRaw(ctx, "_conv/", func(key string, lastModified time.Time, _ int64) bool {
		if lastModified.Before(cutoff) && strings.HasPrefix(key, "_conv/") {
			stale = append(stale, key)
		}
		return len(stale) < 5000 // bound one run's work
	})
	if err != nil {
		slog.Warn("conv cache cleanup: list failed", "error", err)
		return err
	}
	deleted := 0
	for _, key := range stale {
		if ctx.Err() != nil {
			break
		}
		if err := w.deps.Storage.DeleteRaw(ctx, key); err == nil {
			deleted++
		}
	}
	if deleted > 0 {
		slog.Info("conv cache cleanup", "deleted", deleted)
	}
	return nil
}
