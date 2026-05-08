package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type TrendingNewsletterArgs struct{}

func (TrendingNewsletterArgs) Kind() string { return "trending_newsletter" }

type TrendingNewsletterWorker struct {
	river.WorkerDefaults[TrendingNewsletterArgs]
	deps Deps
}

func (w *TrendingNewsletterWorker) Work(ctx context.Context, job *river.Job[TrendingNewsletterArgs]) error {
	slog.Info("trending newsletter started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: build trending schematics digest
	// Send to confirmed subscribers based on frequency (daily/weekly)
	slog.Info("trending newsletter completed")
	return nil
}
