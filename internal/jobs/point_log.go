package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// PointLogArgs are the arguments for the point log reconciliation job.
type PointLogArgs struct{}

func (PointLogArgs) Kind() string { return "point_log_reconcile" }

// PointLogWorker reconciles user point totals against the point_log table.
type PointLogWorker struct {
	river.WorkerDefaults[PointLogArgs]
	deps Deps
}

func (w *PointLogWorker) Work(ctx context.Context, job *river.Job[PointLogArgs]) error {
	slog.Info("reconciling point logs")
	if w.deps.PointLog == nil {
		slog.Warn("point log reconciliation skipped: missing dependencies")
		return nil
	}

	w.deps.PointLog.RecalculateAll()
	slog.Info("point log reconciliation complete")
	return nil
}
