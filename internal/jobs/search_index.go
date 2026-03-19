package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"
	"createmod/internal/search"

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

	// Sync Meilisearch indexes if client is available.
	if w.deps.MeiliClient != nil {
		w.syncMeiliIndexes()
	}

	slog.Info("search index rebuild complete", "count", len(mapped))
	return nil
}

// syncMeiliIndexes converts the in-memory index to Meilisearch documents
// and syncs all three Meilisearch indexes.
func (w *SearchIndexWorker) syncMeiliIndexes() {
	filterIndex := w.deps.Search.GetIndex()
	if len(filterIndex) == 0 {
		return
	}

	// Build Meilisearch documents from the filter index.
	// We don't have the cache entries here, so we re-derive AIDescription
	// from the filter index (it's already been built into the Bleve index).
	docs := search.MapToMeiliDocuments(filterIndex, nil)

	for _, indexUID := range []string{search.MeiliIndexBase, search.MeiliIndexAI, search.MeiliIndexFull} {
		if err := search.SyncMeiliIndex(w.deps.MeiliClient, indexUID, docs); err != nil {
			slog.Error("meili sync failed", "index", indexUID, "error", err)
		} else {
			slog.Info("meili sync complete", "index", indexUID, "docs", len(docs))
		}
	}
}
