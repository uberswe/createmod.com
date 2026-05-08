package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type ReferenceMetadataArgs struct{}

func (ReferenceMetadataArgs) Kind() string { return "reference_metadata_fetch" }

type ReferenceMetadataWorker struct {
	river.WorkerDefaults[ReferenceMetadataArgs]
	deps Deps
}

func (w *ReferenceMetadataWorker) Work(ctx context.Context, job *river.Job[ReferenceMetadataArgs]) error {
	slog.Info("reference metadata fetch started")
	if w.deps.Store == nil {
		return nil
	}

	stale, err := w.deps.Store.References.ListStale(ctx, 50)
	if err != nil {
		return err
	}

	if len(stale) == 0 {
		slog.Info("reference metadata: no stale references")
		return nil
	}

	// TODO: fetch OG metadata (title, thumbnail) from each URL
	// Use net/http + html parsing for og:title, og:image
	slog.Info("reference metadata fetch completed", "checked", len(stale))
	return nil
}
