package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type AnalyticsCleanupArgs struct{}

func (AnalyticsCleanupArgs) Kind() string { return "analytics_cleanup" }

type AnalyticsCleanupWorker struct {
	river.WorkerDefaults[AnalyticsCleanupArgs]
	deps Deps
}

func (w *AnalyticsCleanupWorker) Work(ctx context.Context, job *river.Job[AnalyticsCleanupArgs]) error {
	if w.deps.Store == nil {
		slog.Warn("analytics cleanup skipped: no store")
		return nil
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -30)
	deleted, err := w.deps.Store.Stats.DeleteOldEvents(ctx, cutoff)
	if err != nil {
		slog.Error("analytics cleanup failed", "error", err)
		return err
	}

	slog.Info("analytics cleanup complete", "deleted", deleted)
	return nil
}
