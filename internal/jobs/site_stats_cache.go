package jobs

import (
	"context"
	"log/slog"
	"time"

	"createmod/internal/pages"

	"github.com/riverqueue/river"
)

type SiteStatsCacheArgs struct{}

func (SiteStatsCacheArgs) Kind() string { return "site_stats_cache" }

type SiteStatsCacheWorker struct {
	river.WorkerDefaults[SiteStatsCacheArgs]
	deps Deps
}

func (w *SiteStatsCacheWorker) Work(ctx context.Context, job *river.Job[SiteStatsCacheArgs]) error {
	if w.deps.Store == nil || w.deps.Cache == nil {
		return nil
	}

	start := time.Now()
	for _, window := range []string{"7d", "30d"} {
		pages.WarmSearchStatsCache(ctx, w.deps.Cache, w.deps.Store, window)
	}
	slog.Info("site stats cache warmed", "duration", time.Since(start).Round(time.Millisecond))
	return nil
}
