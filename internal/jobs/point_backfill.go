package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type PointBackfillArgs struct{}

func (PointBackfillArgs) Kind() string { return "point_backfill" }

type PointBackfillWorker struct {
	river.WorkerDefaults[PointBackfillArgs]
	deps Deps
}

func (w *PointBackfillWorker) Work(ctx context.Context, job *river.Job[PointBackfillArgs]) error {
	slog.Info("point backfill started")
	if w.deps.Store == nil || w.deps.PointLog == nil {
		return nil
	}

	const pageSize = 100
	offset := 0
	count := 0

	for {
		users, err := w.deps.Store.Users.ListUsers(ctx, pageSize, offset)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			break
		}

		for _, u := range users {
			w.deps.PointLog.RecalculateUser(u.ID)
			count++
		}

		if len(users) < pageSize {
			break
		}
		offset += pageSize
	}

	slog.Info("point backfill completed", "users", count)
	return nil
}
