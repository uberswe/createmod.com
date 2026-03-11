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

// TempUploadCleanupWorker purges unclaimed temporary uploads older than 30 days
// from both S3 storage and PostgreSQL.
type TempUploadCleanupWorker struct {
	river.WorkerDefaults[TempUploadCleanupArgs]
	deps Deps
}

const cleanupBatchSize = 100

func (w *TempUploadCleanupWorker) Work(ctx context.Context, job *river.Job[TempUploadCleanupArgs]) error {
	if w.deps.Store == nil {
		slog.Warn("temp upload cleanup skipped: missing store")
		return nil
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	var totalDeleted int64

	for {
		uploads, err := w.deps.Store.TempUploads.ListExpiredUnclaimed(ctx, cutoff, cleanupBatchSize)
		if err != nil {
			slog.Error("temp upload cleanup: failed to list expired uploads", "error", err)
			return err
		}
		if len(uploads) == 0 {
			break
		}

		// Delete S3 files and associated temp_upload_files for each upload
		for _, u := range uploads {
			// Delete associated temp upload files from S3
			if w.deps.Storage != nil {
				files, _ := w.deps.Store.TempUploadFiles.ListByToken(ctx, u.Token)
				for _, f := range files {
					if f.NbtS3Key != "" {
						_ = w.deps.Storage.DeleteRaw(ctx, f.NbtS3Key)
					}
				}
			}
			// Delete temp_upload_files DB records
			_ = w.deps.Store.TempUploadFiles.DeleteByToken(ctx, u.Token)

			// Delete the main upload's S3 files
			if w.deps.Storage != nil {
				if u.NbtS3Key != "" {
					_ = w.deps.Storage.DeleteRaw(ctx, u.NbtS3Key)
				}
				if u.ImageS3Key != "" {
					_ = w.deps.Storage.DeleteRaw(ctx, u.ImageS3Key)
				}
			}
		}

		// Bulk delete the temp upload DB rows
		deleted, err := w.deps.Store.TempUploads.DeleteExpiredUnclaimed(ctx, cutoff)
		if err != nil {
			slog.Error("temp upload cleanup: failed to delete expired uploads", "error", err)
			return err
		}
		totalDeleted += deleted

		if len(uploads) < cleanupBatchSize {
			break
		}
	}

	if totalDeleted > 0 {
		slog.Info("temp upload cleanup: complete", "deleted", totalDeleted)
	}
	return nil
}
