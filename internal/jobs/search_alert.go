package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type SearchAlertCheckArgs struct{}

func (SearchAlertCheckArgs) Kind() string { return "search_alert_check" }

type SearchAlertCheckWorker struct {
	river.WorkerDefaults[SearchAlertCheckArgs]
	deps Deps
}

func (w *SearchAlertCheckWorker) Work(ctx context.Context, job *river.Job[SearchAlertCheckArgs]) error {
	slog.Info("search alert check started")
	if w.deps.Store == nil {
		return nil
	}

	alerts, err := w.deps.Store.SearchAlerts.ListActive(ctx, 100)
	if err != nil {
		return err
	}
	if len(alerts) == 0 {
		return nil
	}

	// TODO: for each alert, run query against Meilisearch
	// Compare with last_notified, send email if new results
	for _, a := range alerts {
		_ = w.deps.Store.SearchAlerts.UpdateLastChecked(ctx, a.ID)
	}

	slog.Info("search alert check completed", "alerts_checked", len(alerts))
	return nil
}
