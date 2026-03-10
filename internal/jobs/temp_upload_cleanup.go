package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// TempUploadCleanupArgs are the arguments for the temp upload cleanup job.
type TempUploadCleanupArgs struct{}

func (TempUploadCleanupArgs) Kind() string { return "temp_upload_cleanup" }

// TempUploadCleanupWorker purges expired temporary uploads from PostgreSQL.
type TempUploadCleanupWorker struct {
	river.WorkerDefaults[TempUploadCleanupArgs]
	deps Deps
}

func (w *TempUploadCleanupWorker) Work(ctx context.Context, job *river.Job[TempUploadCleanupArgs]) error {
	if w.deps.Store == nil || w.deps.Store.TempUploads == nil {
		slog.Warn("temp upload cleanup skipped: missing dependencies")
		return nil
	}

	cutoff := time.Now().Add(-2 * time.Hour)
	n, err := w.deps.Store.TempUploads.DeleteExpired(ctx, cutoff)
	if err != nil {
		slog.Error("failed to purge expired temp uploads", "error", err)
		return err
	}
	if n > 0 {
		slog.Info("purged expired temp uploads", "count", n)
	}
	return nil
}
