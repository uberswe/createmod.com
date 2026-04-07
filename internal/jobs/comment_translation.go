package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// CommentTranslationArgs are the arguments for the comment translation backfill job.
type CommentTranslationArgs struct{}

func (CommentTranslationArgs) Kind() string { return "comment_translation_backfill" }

// CommentTranslationWorker backfills missing translations for existing comments.
type CommentTranslationWorker struct {
	river.WorkerDefaults[CommentTranslationArgs]
	deps Deps
}

func (w *CommentTranslationWorker) Timeout(job *river.Job[CommentTranslationArgs]) time.Duration {
	return 30 * time.Minute
}

func (w *CommentTranslationWorker) Work(ctx context.Context, job *river.Job[CommentTranslationArgs]) error {
	slog.Info("backfilling comment translations")
	if w.deps.Translation == nil {
		slog.Warn("comment translation backfill skipped: missing dependencies")
		return nil
	}

	w.deps.Translation.BackfillCommentTranslations()
	slog.Info("comment translation backfill complete")
	return nil
}
