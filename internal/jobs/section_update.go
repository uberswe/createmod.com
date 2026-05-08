package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type SectionUpdateArgs struct{}

func (SectionUpdateArgs) Kind() string { return "section_update" }

type SectionUpdateWorker struct {
	river.WorkerDefaults[SectionUpdateArgs]
	deps Deps
}

func (w *SectionUpdateWorker) Work(ctx context.Context, job *river.Job[SectionUpdateArgs]) error {
	slog.Info("section update digest started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: check section subscriptions and send digest emails
	slog.Info("section update digest completed")
	return nil
}
