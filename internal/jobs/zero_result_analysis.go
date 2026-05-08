package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type ZeroResultAnalysisArgs struct{}

func (ZeroResultAnalysisArgs) Kind() string { return "zero_result_analysis" }

type ZeroResultAnalysisWorker struct {
	river.WorkerDefaults[ZeroResultAnalysisArgs]
	deps Deps
}

func (w *ZeroResultAnalysisWorker) Work(ctx context.Context, job *river.Job[ZeroResultAnalysisArgs]) error {
	slog.Info("zero result analysis started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: analyze zero-result queries from search tracking
	// Generate suggestions using Levenshtein distance against successful queries
	// Upsert into zero_result_suggestions
	slog.Info("zero result analysis completed")
	return nil
}
