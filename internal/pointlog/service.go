package pointlog

import (
	"context"
	"log/slog"
	"time"

	"createmod/internal/store"
)

// pointDef maps an achievement key to the points awarded and a human description.
type pointDef struct {
	Points      int
	Description string
}

// achievementPoints defines points per achievement reason.
var achievementPoints = map[string]pointDef{
	"first_upload":       {Points: 50, Description: "Uploaded your first schematic"},
	"first_upload_bonus": {Points: 30, Description: "First upload bonus"},
	"first_comment":      {Points: 10, Description: "Posted your first comment"},
	// first_guide and first_collection are achievement-only, no separate points.
}

// Service manages the point log background recalculation.
type Service struct {
	appStore *store.Store
	stopChan chan struct{}
}

// New creates a new point log service.
func New(appStore *store.Store) *Service {
	return &Service{
		appStore: appStore,
		stopChan: make(chan struct{}),
	}
}

// Stop signals the background scheduler to stop.
func (s *Service) Stop() {
	close(s.stopChan)
}

// StartScheduler runs RecalculateAll immediately, then every hour.
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

// RecalculateAll finds all users and backfills their point log.
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

// RecalculateUser backfills point_log entries for a single user based on their achievements,
// and corrects the user.points total if needed.
func (s *Service) RecalculateUser(userID string) {
	ctx := context.Background()

	// Load achievements for this user
	achs, err := s.appStore.Achievements.ListUserAchievements(ctx, userID)
	if err != nil {
		return
	}

	// Load existing point_log entries once to avoid repeated queries
	existingEntries, err := s.appStore.Achievements.GetPointLog(ctx, userID)
	if err != nil {
		existingEntries = nil
	}
	loggedReasons := make(map[string]bool, len(existingEntries))
	for _, e := range existingEntries {
		loggedReasons[e.Reason] = true
	}

	for _, ach := range achs {
		def, ok := achievementPoints[ach.Key]
		if !ok || def.Points <= 0 {
			continue
		}

		// Check if point_log entry already exists for this reason
		if loggedReasons[ach.Key] {
			continue
		}

		// Create the missing entry
		_ = s.appStore.Achievements.CreatePointLog(ctx, &store.PointLogEntry{
			UserID:      userID,
			Points:      def.Points,
			Reason:      ach.Key,
			Description: def.Description,
			EarnedAt:    time.Now(),
		})
	}

	// Sum all point_log entries and reconcile user.points
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

// CreateLogEntry creates a point_log entry with earned_at = now.
// Call this from hooks when awarding points in real-time.
func CreateLogEntry(appStore *store.Store, userID string, points int, reason, description string) {
	createLogEntryDirect(appStore, userID, points, reason, description, time.Now())
}

func createLogEntryDirect(appStore *store.Store, userID string, points int, reason, description string, earnedAt time.Time) {
	ctx := context.Background()
	_ = appStore.Achievements.CreatePointLog(ctx, &store.PointLogEntry{
		UserID:      userID,
		Points:      points,
		Reason:      reason,
		Description: description,
		EarnedAt:    earnedAt,
	})
}
