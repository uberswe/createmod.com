package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"
	"createmod/internal/search"
	"createmod/internal/store"

	"github.com/riverqueue/river"
)

// SearchIndexUpsertArgs triggers an incremental Meilisearch upsert for a single schematic.
type SearchIndexUpsertArgs struct {
	SchematicID string `json:"schematic_id"`
}

func (SearchIndexUpsertArgs) Kind() string { return "search_index_upsert" }

// SearchIndexUpsertWorker updates a single schematic in the Meilisearch index.
type SearchIndexUpsertWorker struct {
	river.WorkerDefaults[SearchIndexUpsertArgs]
	deps Deps
}

func (w *SearchIndexUpsertWorker) Work(ctx context.Context, job *river.Job[SearchIndexUpsertArgs]) error {
	id := job.Args.SchematicID
	if id == "" || w.deps.Store == nil || w.deps.MeiliClient == nil {
		return nil
	}

	s, err := w.deps.Store.Schematics.GetByID(ctx, id)
	if err != nil {
		slog.Warn("search upsert: failed to load schematic", "id", id, "error", err)
		return nil
	}

	// Only index publicly visible schematics; delete from index otherwise.
	if s == nil || !store.IsPublicState(s.ModerationState) {
		return w.deleteFromIndex(id)
	}

	// Build a full models.Schematic with categories, tags, views, etc.
	mapped := pages.MapStoreSchematicsNoCache(w.deps.Store, []store.Schematic{*s}, w.deps.Cache)
	if len(mapped) == 0 {
		return nil
	}

	modDisplayNames := make(map[string]string)
	if allMeta, err := w.deps.Store.ModMetadata.ListAll(ctx); err == nil {
		for _, m := range allMeta {
			if m.DisplayName != "" {
				modDisplayNames[m.Namespace] = m.DisplayName
			}
		}
	}

	trendingScores := w.deps.Search.GetTrendingScores()
	doc := search.BuildSingleDocument(mapped[0], modDisplayNames, trendingScores)

	if err := search.SyncMeiliIndex(w.deps.MeiliClient, search.MeiliIndex, []search.MeiliDocument{doc}); err != nil {
		slog.Error("search upsert: sync failed", "id", id, "error", err)
		return err
	}

	slog.Info("search upsert: indexed", "id", id)
	return nil
}

func (w *SearchIndexUpsertWorker) deleteFromIndex(id string) error {
	index := w.deps.MeiliClient.Index(search.MeiliIndex)
	_, err := index.DeleteDocument(id, nil)
	if err != nil {
		slog.Error("search upsert: delete failed", "id", id, "error", err)
		return err
	}
	slog.Info("search upsert: removed from index", "id", id)
	return nil
}

// upsertSchematicToMeili builds and syncs a single schematic to Meilisearch.
// Used by both the upsert worker and other jobs that publish schematics.
func upsertSchematicToMeili(ctx context.Context, deps Deps, schematicID string) {
	s, err := deps.Store.Schematics.GetByID(ctx, schematicID)
	if err != nil || s == nil || !store.IsPublicState(s.ModerationState) {
		return
	}

	mapped := pages.MapStoreSchematicsNoCache(deps.Store, []store.Schematic{*s}, deps.Cache)
	if len(mapped) == 0 {
		return
	}

	modDisplayNames := make(map[string]string)
	if allMeta, err := deps.Store.ModMetadata.ListAll(ctx); err == nil {
		for _, m := range allMeta {
			if m.DisplayName != "" {
				modDisplayNames[m.Namespace] = m.DisplayName
			}
		}
	}

	var trendingScores map[string]float64
	if deps.Search != nil {
		trendingScores = deps.Search.GetTrendingScores()
	}

	doc := search.BuildSingleDocument(mapped[0], modDisplayNames, trendingScores)
	if err := search.SyncMeiliIndex(deps.MeiliClient, search.MeiliIndex, []search.MeiliDocument{doc}); err != nil {
		slog.Error("search upsert: sync failed", "id", schematicID, "error", err)
	}
}

// SearchIndexDeleteArgs triggers removal of a schematic from the Meilisearch index.
type SearchIndexDeleteArgs struct {
	SchematicID string `json:"schematic_id"`
}

func (SearchIndexDeleteArgs) Kind() string { return "search_index_delete" }

// SearchIndexDeleteWorker removes a single schematic from the Meilisearch index.
type SearchIndexDeleteWorker struct {
	river.WorkerDefaults[SearchIndexDeleteArgs]
	deps Deps
}

func (w *SearchIndexDeleteWorker) Work(ctx context.Context, job *river.Job[SearchIndexDeleteArgs]) error {
	id := job.Args.SchematicID
	if id == "" || w.deps.MeiliClient == nil {
		return nil
	}

	index := w.deps.MeiliClient.Index(search.MeiliIndex)
	_, err := index.DeleteDocument(id, nil)
	if err != nil {
		slog.Error("search delete: failed", "id", id, "error", err)
		return err
	}
	slog.Info("search delete: removed from index", "id", id)
	return nil
}
