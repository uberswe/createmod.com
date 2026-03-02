package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// SitemapArgs are the arguments for the sitemap generation job.
type SitemapArgs struct{}

func (SitemapArgs) Kind() string { return "sitemap_generate" }

// SitemapWorker generates XML sitemaps.
type SitemapWorker struct {
	river.WorkerDefaults[SitemapArgs]
	deps Deps
}

func (w *SitemapWorker) Work(ctx context.Context, job *river.Job[SitemapArgs]) error {
	slog.Info("generating sitemaps")
	if w.deps.App == nil || w.deps.Sitemap == nil {
		slog.Warn("sitemap generation skipped: missing dependencies")
		return nil
	}

	w.deps.Sitemap.Generate(w.deps.App)
	slog.Info("sitemap generation complete")
	return nil
}
