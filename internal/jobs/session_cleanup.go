package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// SessionCleanupArgs are the arguments for the session cleanup job.
type SessionCleanupArgs struct{}

func (SessionCleanupArgs) Kind() string { return "session_cleanup" }

// SessionCleanupWorker removes expired sessions from PostgreSQL.
type SessionCleanupWorker struct {
	river.WorkerDefaults[SessionCleanupArgs]
	deps Deps
}

func (w *SessionCleanupWorker) Work(ctx context.Context, job *river.Job[SessionCleanupArgs]) error {
	if w.deps.SessionStore == nil {
		slog.Warn("session cleanup skipped: no session store")
		return nil
	}

	slog.Info("cleaning up expired sessions")
	if err := w.deps.SessionStore.Cleanup(ctx); err != nil {
		slog.Error("session cleanup failed", "error", err)
		return err
	}
	slog.Info("session cleanup complete")
	return nil
}
