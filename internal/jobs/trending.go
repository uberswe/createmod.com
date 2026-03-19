package jobs

import (
	"context"
	"log/slog"

	"createmod/internal/pages"

	"github.com/riverqueue/river"
)

// TrendingArgs are the arguments for the trending score computation job.
type TrendingArgs struct{}

func (TrendingArgs) Kind() string { return "trending_scores" }

// TrendingWorker computes trending scores for schematics.
type TrendingWorker struct {
	river.WorkerDefaults[TrendingArgs]
	deps       Deps
	WindowDays []int
}

func (w *TrendingWorker) Work(ctx context.Context, job *river.Job[TrendingArgs]) error {
	slog.Info("computing trending scores")
	if w.deps.Store == nil || w.deps.Search == nil {
		slog.Warn("trending scores skipped: missing dependencies")
		return nil
	}

	if scores := pages.ComputeTrendingScoresFromStore(w.deps.Store); scores != nil {
		w.deps.Search.SetTrendingScores(scores)
	}

	// Warm caches
	if w.deps.Cache != nil {
		pages.WarmIndexCache(w.deps.Cache, w.deps.Store, w.WindowDays)
		pages.WarmVideosCache(w.deps.Cache, w.deps.Store)
	}

	slog.Info("trending scores updated")
	return nil
}
