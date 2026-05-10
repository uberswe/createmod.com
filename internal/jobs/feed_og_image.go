package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"

	"github.com/riverqueue/river"
)

type FeedOGImageArgs struct{}

func (FeedOGImageArgs) Kind() string { return "feed_og_image" }

type FeedOGImageWorker struct {
	river.WorkerDefaults[FeedOGImageArgs]
	deps Deps
}

func (w *FeedOGImageWorker) Work(ctx context.Context, job *river.Job[FeedOGImageArgs]) error {
	slog.Info("feed OG image generation started")
	if w.deps.Storage == nil || w.deps.Store == nil || w.deps.Cache == nil {
		slog.Warn("feed OG: missing deps (storage, store, or cache)")
		return nil
	}

	pages.GenerateFeedOGImage(w.deps.Storage, w.deps.Store, w.deps.Cache, "trending")
	pages.GenerateFeedOGImage(w.deps.Storage, w.deps.Store, w.deps.Cache, "highest_scores")

	slog.Info("feed OG image generation completed")
	return nil
}
