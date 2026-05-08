package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type BadgeRecalculationArgs struct{}

func (BadgeRecalculationArgs) Kind() string { return "badge_recalculation" }

type BadgeRecalculationWorker struct {
	river.WorkerDefaults[BadgeRecalculationArgs]
	deps Deps
}

func (w *BadgeRecalculationWorker) Work(ctx context.Context, job *river.Job[BadgeRecalculationArgs]) error {
	slog.Info("badge recalculation started")
	if w.deps.Store == nil {
		slog.Warn("badge recalculation skipped: missing store")
		return nil
	}

	badges, err := w.deps.Store.Badges.List(ctx)
	if err != nil {
		return err
	}
	badgeByKey := make(map[string]string, len(badges))
	for _, b := range badges {
		badgeByKey[b.Key] = b.ID
	}

	const pageSize = 500
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
			w.recalculateUserBadges(ctx, u.ID, badgeByKey)
			count++
		}

		if len(users) < pageSize {
			break
		}
		offset += pageSize
	}

	slog.Info("badge recalculation completed", "users", count)
	return nil
}

func (w *BadgeRecalculationWorker) recalculateUserBadges(ctx context.Context, userID string, badgeByKey map[string]string) {
	schematics, err := w.deps.Store.Schematics.ListByAuthorAll(ctx, userID, 10000, 0)
	if err != nil {
		return
	}

	ids := make([]string, len(schematics))
	for i, s := range schematics {
		ids[i] = s.ID
	}

	totalViews := 0
	if len(ids) > 0 {
		viewCounts, err := w.deps.Store.ViewRatings.BatchGetViewCounts(ctx, ids)
		if err == nil {
			for _, v := range viewCounts {
				totalViews += v
			}
		}
	}

	ratingCount := 0
	if len(ids) > 0 {
		ratings, err := w.deps.Store.ViewRatings.BatchGetRatings(ctx, ids)
		if err == nil {
			for _, r := range ratings {
				ratingCount += r.RatingCount
			}
		}
	}

	commentCount, _ := w.deps.Store.Comments.CountByUser(ctx, userID)

	type milestone struct {
		key       string
		threshold int
		current   int
	}
	milestones := []milestone{
		{"views_10k", 10000, totalViews},
		{"views_100k", 100000, totalViews},
		{"views_1m", 1000000, totalViews},
		{"ratings_100", 100, ratingCount},
		{"comments_100", 100, int(commentCount)},
	}

	for _, m := range milestones {
		badgeID, ok := badgeByKey[m.key]
		if !ok {
			continue
		}
		if m.current >= m.threshold {
			_ = w.deps.Store.Badges.AwardBadge(ctx, userID, badgeID)
		}
	}
}
