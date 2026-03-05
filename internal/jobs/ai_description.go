package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// AIDescriptionArgs are the arguments for the AI description generation job.
type AIDescriptionArgs struct{}

func (AIDescriptionArgs) Kind() string { return "ai_description" }

// AIDescriptionWorker generates AI descriptions for schematics that lack them.
type AIDescriptionWorker struct {
	river.WorkerDefaults[AIDescriptionArgs]
	deps Deps
}

func (w *AIDescriptionWorker) Work(ctx context.Context, job *river.Job[AIDescriptionArgs]) error {
	slog.Info("generating AI descriptions")
	if w.deps.AIDesc == nil || w.deps.Storage == nil || w.deps.Store == nil {
		slog.Warn("AI description generation skipped: missing dependencies")
		return nil
	}

	w.deps.AIDesc.ProcessSchematics(w.deps.Storage, w.deps.Store)
	slog.Info("AI description generation complete")
	return nil
}
