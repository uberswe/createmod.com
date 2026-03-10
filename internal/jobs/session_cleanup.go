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

	// Clean up expired download tokens
	if w.deps.Store != nil && w.deps.Store.DownloadTokens != nil {
		if err := w.deps.Store.DownloadTokens.CleanupExpired(ctx); err != nil {
			slog.Error("download token cleanup failed", "error", err)
		} else {
			slog.Info("download token cleanup complete")
		}
	}

	slog.Info("session cleanup complete")
	return nil
}
