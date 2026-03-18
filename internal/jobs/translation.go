package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// TranslationArgs are the arguments for the translation backfill job.
type TranslationArgs struct{}

func (TranslationArgs) Kind() string { return "translation_backfill" }

// TranslationWorker fills in missing translations for schematics, guides, and collections.
type TranslationWorker struct {
	river.WorkerDefaults[TranslationArgs]
	deps Deps
}

func (w *TranslationWorker) Timeout(job *river.Job[TranslationArgs]) time.Duration {
	return 30 * time.Minute
}

func (w *TranslationWorker) Work(ctx context.Context, job *river.Job[TranslationArgs]) error {
	slog.Info("backfilling translations")
	if w.deps.Translation == nil {
		slog.Warn("translation backfill skipped: missing dependencies")
		return nil
	}

	w.deps.Translation.BackfillMissingTranslations()
	slog.Info("translation backfill complete")
	return nil
}
