package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// SearchCleanupArgs are the arguments for the search cleanup job.
type SearchCleanupArgs struct{}

func (SearchCleanupArgs) Kind() string { return "search_cleanup" }

// SearchCleanupWorker prunes single-use searches older than 90 days
// and refreshes the search_query_counts materialized view.
type SearchCleanupWorker struct {
	river.WorkerDefaults[SearchCleanupArgs]
	deps Deps
}

func (w *SearchCleanupWorker) Work(ctx context.Context, job *river.Job[SearchCleanupArgs]) error {
	if w.deps.Store == nil {
		slog.Warn("search cleanup skipped: no store")
		return nil
	}

	slog.Info("pruning old single-use searches")
	deleted, err := w.deps.Store.SearchTracking.PruneOldSearches(ctx)
	if err != nil {
		slog.Error("search cleanup: prune failed", "error", err)
		return err
	}
	slog.Info("search cleanup: pruned old searches", "deleted", deleted)

	return nil
}
