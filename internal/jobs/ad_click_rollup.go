package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type AdClickRollupArgs struct{}

func (AdClickRollupArgs) Kind() string { return "ad_click_rollup" }

type AdClickRollupWorker struct {
	river.WorkerDefaults[AdClickRollupArgs]
	deps Deps
}

func (w *AdClickRollupWorker) Work(ctx context.Context, job *river.Job[AdClickRollupArgs]) error {
	if w.deps.Store == nil {
		slog.Warn("ad click rollup skipped: no store")
		return nil
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -30).Format("20060102")
	err := w.deps.Store.AdClicks.RollupAndClean(ctx, cutoff)
	if err != nil {
		slog.Error("ad click rollup failed", "error", err)
		return err
	}

	slog.Info("ad click rollup complete", "cutoff", cutoff)
	return nil
}
