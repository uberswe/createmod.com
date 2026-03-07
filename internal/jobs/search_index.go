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

	storeSchematics, err := w.deps.Store.Schematics.ListAllForIndex(ctx)
	if err != nil {
		slog.Error("search index rebuild: failed to load schematics", "error", err)
		return err
	}
	mapped := pages.MapStoreSchematics(w.deps.Store, storeSchematics, w.deps.Cache)
	w.deps.Search.BuildIndex(mapped)
	slog.Info("search index rebuild complete", "count", len(mapped))
	return nil
}
