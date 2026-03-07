package pages

import (
	"context"
	"createmod/internal/store"
	"log/slog"
)

// awardFirstGuide awards the first_guide achievement if this is the user's first guide.
func awardFirstGuide(appStore *store.Store, userID string) {
	ctx := context.Background()
	count, err := appStore.Guides.CountByUser(ctx, userID)
	if err != nil || count != 1 {
		return
	}
	ach, err := appStore.Achievements.GetByKey(ctx, "first_guide")
	if err != nil || ach == nil {
		return
	}
	has, _ := appStore.Achievements.HasAchievement(ctx, userID, ach.ID)
	if has {
		return
	}
	if err := appStore.Achievements.Award(ctx, userID, ach.ID); err != nil {
		slog.Warn("failed to award first_guide achievement", "error", err, "user", userID)
	}
}

// awardFirstCollection awards the first_collection achievement if this is the user's first collection.
func awardFirstCollection(appStore *store.Store, userID string) {
	ctx := context.Background()
	count, err := appStore.Collections.CountByUser(ctx, userID)
	if err != nil || count != 1 {
		return
	}
	ach, err := appStore.Achievements.GetByKey(ctx, "first_collection")
	if err != nil || ach == nil {
		return
	}
	has, _ := appStore.Achievements.HasAchievement(ctx, userID, ach.ID)
	if has {
		return
	}
	if err := appStore.Achievements.Award(ctx, userID, ach.ID); err != nil {
		slog.Warn("failed to award first_collection achievement", "error", err, "user", userID)
	}
}
