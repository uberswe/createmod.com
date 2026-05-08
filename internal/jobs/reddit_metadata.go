package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type RedditMetadataArgs struct{}

func (RedditMetadataArgs) Kind() string { return "reddit_metadata_refresh" }

type RedditMetadataWorker struct {
	river.WorkerDefaults[RedditMetadataArgs]
	deps Deps
}

func (w *RedditMetadataWorker) Work(ctx context.Context, job *river.Job[RedditMetadataArgs]) error {
	slog.Info("reddit metadata refresh started")
	if w.deps.Store == nil {
		return nil
	}

	stale, err := w.deps.Store.RedditLinks.ListStale(ctx, 50)
	if err != nil {
		return err
	}

	if len(stale) == 0 {
		slog.Info("reddit metadata: no stale links")
		return nil
	}

	// TODO: fetch metadata from Reddit JSON API for each stale link
	// For each link, GET https://www.reddit.com/{path}.json and parse
	slog.Info("reddit metadata refresh completed", "checked", len(stale))
	return nil
}
