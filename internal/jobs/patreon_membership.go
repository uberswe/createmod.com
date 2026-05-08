package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type PatreonMembershipArgs struct{}

func (PatreonMembershipArgs) Kind() string { return "patreon_membership_check" }

type PatreonMembershipWorker struct {
	river.WorkerDefaults[PatreonMembershipArgs]
	deps Deps
}

func (w *PatreonMembershipWorker) Work(ctx context.Context, job *river.Job[PatreonMembershipArgs]) error {
	slog.Info("patreon membership check started")
	if w.deps.Store == nil {
		return nil
	}

	// TODO: for each Patreon-linked user, check membership via API
	// Award/revoke patreon_creator and patreon_mod badges
	slog.Info("patreon membership check completed")
	return nil
}
