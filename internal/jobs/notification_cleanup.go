package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

type NotificationCleanupArgs struct{}

func (NotificationCleanupArgs) Kind() string { return "notification_cleanup" }

type NotificationCleanupWorker struct {
	river.WorkerDefaults[NotificationCleanupArgs]
	deps Deps
}

func (w *NotificationCleanupWorker) Work(ctx context.Context, job *river.Job[NotificationCleanupArgs]) error {
	slog.Info("notification cleanup started")
	if w.deps.Store == nil {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -90)
	if err := w.deps.Store.Notifications.DeleteOld(ctx, cutoff); err != nil {
		slog.Error("notification cleanup failed", "error", err)
		return err
	}

	slog.Info("notification cleanup completed")
	return nil
}
