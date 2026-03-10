package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"

	"github.com/riverqueue/river"
)

// SearchIndexArgs are the arguments for the search index rebuild job.
type SearchIndexArgs struct{}

func (SearchIndexArgs) Kind() string { return "search_index_rebuild" }

// SearchIndexWorker rebuilds the Bleve search index.
type SearchIndexWorker struct {
	river.WorkerDefaults[SearchIndexArgs]
	deps Deps
}

func (w *SearchIndexWorker) Work(ctx context.Context, job *river.Job[SearchIndexArgs]) error {
	slog.Info("rebuilding search index")
	if w.deps.Store == nil || w.deps.Search == nil {
		slog.Warn("search index rebuild skipped: missing dependencies")
		return nil
	}

	// On the very first run, try to load the S3 cache so searches work
	// while the full DB rebuild is in progress.
	w.deps.Search.WarmFromStorage()

	storeSchematics, err := w.deps.Store.Schematics.ListAllForIndex(ctx)
	if err != nil {
		slog.Error("search index rebuild: failed to load schematics", "error", err)
		return err
	}
	mapped := pages.MapStoreSchematics(w.deps.Store, storeSchematics, w.deps.Cache)
	w.deps.Search.BuildIndex(mapped)

	// Also refresh trending scores so they're available right after index rebuild.
	if scores := pages.ComputeTrendingScoresFromStore(w.deps.Store); scores != nil {
		w.deps.Search.SetTrendingScores(scores)
	}

	slog.Info("search index rebuild complete", "count", len(mapped))
	return nil
}
