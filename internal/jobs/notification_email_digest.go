package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type NotificationEmailDigestArgs struct{}

func (NotificationEmailDigestArgs) Kind() string { return "notification_email_digest" }

type NotificationEmailDigestWorker struct {
	river.WorkerDefaults[NotificationEmailDigestArgs]
	deps Deps
}

func (w *NotificationEmailDigestWorker) Work(ctx context.Context, job *river.Job[NotificationEmailDigestArgs]) error {
	slog.Info("notification email digest started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: batch unread notifications by user preference frequency
	// Send email digests for users with daily/weekly email notifications
	slog.Info("notification email digest completed")
	return nil
}
