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
	if w.deps.Store == nil || w.deps.Sitemap == nil {
		slog.Warn("sitemap generation skipped: missing dependencies")
		return nil
	}

	w.deps.Sitemap.Generate(w.deps.Store)

	// Invalidate the RSS feed cache so the next request rebuilds it with fresh data.
	if w.deps.Cache != nil {
		w.deps.Cache.Delete("rss_feed")
	}

	slog.Info("sitemap generation complete")
	return nil
}
