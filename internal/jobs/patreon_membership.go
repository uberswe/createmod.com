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

	patreonUsers, err := w.deps.Store.Auth.ListByProvider(ctx, "patreon")
	if err != nil {
		return err
	}

	if len(patreonUsers) == 0 {
		slog.Info("patreon membership: no linked users")
		return nil
	}

	badge, err := w.deps.Store.Badges.GetByKey(ctx, "patreon_creator")
	if err != nil {
		slog.Warn("patreon membership: badge not found", "key", "patreon_creator", "err", err)
		return nil
	}

	awarded := 0
	for _, ea := range patreonUsers {
		if err := w.deps.Store.Badges.AwardBadge(ctx, ea.UserID, badge.ID); err != nil {
			slog.Warn("patreon membership: award failed", "user", ea.UserID, "err", err)
		} else {
			awarded++
		}
	}

	slog.Info("patreon membership check completed", "linked_users", len(patreonUsers), "awarded", awarded)
	return nil
}
