package pointlog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"createmod/internal/store"
)

const (
	ReasonUpload          = "upload"
	ReasonComment         = "comment"
	ReasonRating4         = "rating_4plus"
	ReasonDown100         = "downloads_100"
	ReasonViews10K        = "views_10k"
	ReasonFirstComm       = "first_comment"
	ReasonViews100        = "views_100_milestone"
	ReasonViews1K         = "views_1k_milestone"
	ReasonViews10KMilestone = "views_10k_milestone"
)

type pointRule struct {
	Points      int
	Description string
}

var rules = map[string]pointRule{
	ReasonUpload:            {Points: 1, Description: "Uploaded a schematic"},
	ReasonComment:           {Points: 1, Description: "Commented on a schematic"},
	ReasonRating4:           {Points: 2, Description: "Schematic received 4+ star rating"},
	ReasonDown100:           {Points: 2, Description: "Schematic reached 100 downloads"},
	ReasonViews10K:          {Points: 5, Description: "Schematic reached 10,000 views"},
	ReasonFirstComm:         {Points: 10, Description: "Posted your first comment"},
	ReasonViews100:          {Points: 5, Description: "Schematic reached 100 views"},
	ReasonViews1K:           {Points: 25, Description: "Schematic reached 1,000 views"},
	ReasonViews10KMilestone: {Points: 100, Description: "Schematic reached 10,000 views (milestone)"},
}

type Service struct {
	appStore *store.Store
	stopChan chan struct{}
}

func New(appStore *store.Store) *Service {
	return &Service{
		appStore: appStore,
		stopChan: make(chan struct{}),
	}
}

func (s *Service) Stop() {
	close(s.stopChan)
}

func (s *Service) StartScheduler() {
	go func() {
		s.RecalculateAll()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.RecalculateAll()
			case <-s.stopChan:
				slog.Info("Point log scheduler stopped")
				return
			}
		}
	}()
	slog.Info("Point log scheduler started (polling every hour)")
}

func (s *Service) RecalculateAll() {
	slog.Info("Point log recalculation started")

	ctx := context.Background()
	const pageSize = 500
	offset := 0
	count := 0

	for {
		users, err := s.appStore.Users.ListUsers(ctx, pageSize, offset)
		if err != nil {
			slog.Error("point_log: failed to list users", "error", err)
			return
		}
		if len(users) == 0 {
			break
		}

		for _, u := range users {
			s.RecalculateUser(u.ID)
			count++
		}

		if len(users) < pageSize {
			break
		}
		offset += pageSize
	}

	slog.Info("Point log recalculation completed", "users", count)
}

func (s *Service) RecalculateUser(userID string) {
	ctx := context.Background()
	now := time.Now()

	schematics, err := s.appStore.Schematics.ListByAuthorAll(ctx, userID, 10000, 0)
	if err != nil {
		return
	}

	schematicIDs := make([]string, len(schematics))
	for i, sc := range schematics {
		schematicIDs[i] = sc.ID
	}

	for _, sc := range schematics {
		awardPoint(ctx, s.appStore, userID, ReasonUpload, sc.ID, now)
	}

	if len(schematicIDs) > 0 {
		viewCounts, _ := s.appStore.ViewRatings.BatchGetViewCounts(ctx, schematicIDs)
		downloadCounts, _ := s.appStore.ViewRatings.BatchGetDownloadCounts(ctx, schematicIDs)
		ratings, _ := s.appStore.ViewRatings.BatchGetRatings(ctx, schematicIDs)

		for _, sc := range schematics {
			if views, ok := viewCounts[sc.ID]; ok {
				// Repeating: +5 for every 10,000 views
				milestones := views / 10000
				for i := 1; i <= milestones; i++ {
					ref := fmt.Sprintf("%s:%d", sc.ID, i*10000)
					awardPoint(ctx, s.appStore, userID, ReasonViews10K, ref, now)
				}
				// One-time view milestones
				if views >= 100 {
					awardPoint(ctx, s.appStore, userID, ReasonViews100, sc.ID, now)
				}
				if views >= 1000 {
					awardPoint(ctx, s.appStore, userID, ReasonViews1K, sc.ID, now)
				}
				if views >= 10000 {
					awardPoint(ctx, s.appStore, userID, ReasonViews10KMilestone, sc.ID, now)
				}
			}
			if downloads, ok := downloadCounts[sc.ID]; ok {
				// Repeating: +2 for every 100 downloads
				milestones := downloads / 100
				for i := 1; i <= milestones; i++ {
					ref := fmt.Sprintf("%s:%d", sc.ID, i*100)
					awardPoint(ctx, s.appStore, userID, ReasonDown100, ref, now)
				}
			}
			if r, ok := ratings[sc.ID]; ok && r.AvgRating >= 4.0 && r.RatingCount > 0 {
				awardPoint(ctx, s.appStore, userID, ReasonRating4, sc.ID, now)
			}
		}
	}

	commentCount, err := s.appStore.Comments.CountByUser(ctx, userID)
	if err == nil && commentCount > 0 {
		for i := int64(0); i < commentCount; i++ {
			awardPoint(ctx, s.appStore, userID, ReasonComment, fmt.Sprintf("comment_%d", i), now)
		}
		awardPoint(ctx, s.appStore, userID, ReasonFirstComm, "first", now)
	}

	total, err := s.appStore.Achievements.SumUserPoints(ctx, userID)
	if err != nil {
		return
	}

	u, err := s.appStore.Users.GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	if u.Points != total {
		_ = s.appStore.Users.UpdateUserPoints(ctx, userID, total)
	}
}

func awardPoint(ctx context.Context, appStore *store.Store, userID, reason, referenceID string, earnedAt time.Time) {
	rule, ok := rules[reason]
	if !ok {
		return
	}
	_ = appStore.Achievements.CreatePointLog(ctx, &store.PointLogEntry{
		UserID:      userID,
		Points:      rule.Points,
		Reason:      reason,
		ReferenceID: referenceID,
		Description: rule.Description,
		EarnedAt:    earnedAt,
	})
}

func CreateLogEntry(appStore *store.Store, userID string, points int, reason, referenceID, description string) {
	ctx := context.Background()
	_ = appStore.Achievements.CreatePointLog(ctx, &store.PointLogEntry{
		UserID:      userID,
		Points:      points,
		Reason:      reason,
		ReferenceID: referenceID,
		Description: description,
		EarnedAt:    time.Now(),
	})
}
