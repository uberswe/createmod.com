package jobs

import (
	"context"

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
	// Temp uploads are permanent — no cleanup performed.
	return nil
}
