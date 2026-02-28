package pointlog

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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
	stopChan chan struct{}
}

// New creates a new point log service.
func New() *Service {
	return &Service{
		stopChan: make(chan struct{}),
	}
}

// Stop signals the background scheduler to stop.
func (s *Service) Stop() {
	close(s.stopChan)
}

// StartScheduler runs RecalculateAll immediately, then every hour.
func (s *Service) StartScheduler(app *pocketbase.PocketBase) {
	go func() {
		RecalculateAll(app)

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				RecalculateAll(app)
			case <-s.stopChan:
				app.Logger().Info("Point log scheduler stopped")
				return
			}
		}
	}()
	app.Logger().Info("Point log scheduler started (polling every hour)")
}

// RecalculateAll finds all users with achievements and backfills their point log.
func RecalculateAll(app *pocketbase.PocketBase) {
	app.Logger().Info("Point log recalculation started")

	uaColl, err := app.FindCollectionByNameOrId("user_achievements")
	if err != nil {
		app.Logger().Debug("point_log: user_achievements collection not found", "error", err)
		return
	}

	// Find all user_achievements records (limit 10000)
	uas, err := app.FindRecordsByFilter(uaColl.Id, "1=1", "-created", 10000, 0)
	if err != nil {
		app.Logger().Debug("point_log: failed to query user_achievements", "error", err)
		return
	}

	// Collect distinct user IDs
	userSet := make(map[string]struct{})
	for _, ua := range uas {
		uid := ua.GetString("user")
		if uid != "" {
			userSet[uid] = struct{}{}
		}
	}

	for uid := range userSet {
		RecalculateUser(app, uid)
	}

	app.Logger().Info("Point log recalculation completed", "users", len(userSet))
}

// RecalculateUser backfills point_log entries for a single user based on their achievements,
// and corrects the user.points total if needed.
func RecalculateUser(app *pocketbase.PocketBase, userID string) {
	achColl, err := app.FindCollectionByNameOrId("achievements")
	if err != nil {
		return
	}
	uaColl, err := app.FindCollectionByNameOrId("user_achievements")
	if err != nil {
		return
	}

	// Load user_achievements for this user
	uas, err := app.FindRecordsByFilter(uaColl.Id, "user = {:u}", "-created", 100, 0, dbx.Params{"u": userID})
	if err != nil {
		return
	}

	for _, ua := range uas {
		achID := ua.GetString("achievement")
		if achID == "" {
			continue
		}
		achRec, err := app.FindRecordById(achColl.Id, achID)
		if err != nil {
			continue
		}
		key := achRec.GetString("key")
		def, ok := achievementPoints[key]
		if !ok || def.Points <= 0 {
			continue
		}

		// Check if point_log entry already exists
		existing, _ := app.FindRecordsByFilter("point_log", "user = {:u} && reason = {:r}", "-created", 1, 0, dbx.Params{"u": userID, "r": key})
		if len(existing) > 0 {
			continue
		}

		// Create the missing entry with earned_at = user_achievement.created
		earnedAt := ua.GetDateTime("created").Time()
		if earnedAt.IsZero() {
			earnedAt = time.Now()
		}
		createLogEntryDirect(app, userID, def.Points, key, def.Description, earnedAt)
	}

	// Sum all point_log entries and reconcile user.points
	plRecs, err := app.FindRecordsByFilter("point_log", "user = {:u}", "-earned_at", 10000, 0, dbx.Params{"u": userID})
	if err != nil {
		return
	}
	total := 0
	for _, pl := range plRecs {
		total += pl.GetInt("points")
	}

	u, err := app.FindRecordById("_pb_users_auth_", userID)
	if err != nil {
		return
	}
	if u.GetInt("points") != total {
		u.Set("points", total)
		_ = app.Save(u)
	}
}

// CreateLogEntry creates a point_log entry with earned_at = now.
// Call this from hooks when awarding points in real-time.
func CreateLogEntry(app core.App, userID string, points int, reason, description string) {
	createLogEntryDirect(app, userID, points, reason, description, time.Now())
}

func createLogEntryDirect(app core.App, userID string, points int, reason, description string, earnedAt time.Time) {
	coll, err := app.FindCollectionByNameOrId("point_log")
	if err != nil {
		return
	}
	rec := core.NewRecord(coll)
	rec.Set("user", userID)
	rec.Set("points", points)
	rec.Set("reason", reason)
	rec.Set("description", description)
	rec.Set("earned_at", earnedAt)
	_ = app.Save(rec)
}
