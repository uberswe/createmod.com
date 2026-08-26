package pages

import (
	"context"
	"net/http"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
)

var adminUserTemplates = append([]string{
	"./template/admin_user.html",
}, commonTemplates...)

// AdminUserViolationRow is one moderation violation on a user's upload.
type AdminUserViolationRow struct {
	When           time.Time
	SchematicTitle string
	SchematicName  string
	Kind           string // rejected | flagged | changes | limited | removed | state
	KindLabel      string
	ActorType      string
	ActorUsername  string
	Reason         string
}

// AdminUserData powers the admin user page: profile summary + a log of every
// moderation violation across the user's uploads. (#1646)
type AdminUserData struct {
	DefaultData
	ProfileUsername string
	ProfileID       string
	ProfileEmail    string
	ProfileAvatar   string
	JoinedAt        time.Time
	IsBanned        bool
	PublishedCount  int64
	RemovedCount    int64
	ViolationCount  int
	Violations      []AdminUserViolationRow
}

// violationKind maps a moderation-log entry to a display kind + label.
func violationKind(action, newState string) (kind, label string) {
	if action == "soft_delete" || newState == store.ModerationDeleted {
		return "removed", "Removed"
	}
	switch newState {
	case store.ModerationRejected, store.ModerationRejectedFinal:
		return "rejected", "Rejected (final)"
	case store.ModerationRejectedFixable:
		return "rejected", "Rejected (fixable)"
	case store.ModerationFlagged:
		return "flagged", "Flagged for review"
	case store.ModerationChangesRequested:
		return "changes", "Changes requested"
	case store.ModerationPublishedLimited:
		return "limited", "Published with limits"
	}
	return "state", newState
}

// AdminUserHandler renders GET /admin/user/{username}: the user's profile plus a
// log of every moderation violation across their uploads. (#1646)
func AdminUserHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()
		username := strings.TrimSpace(e.Request.PathValue("username"))
		if username == "" {
			return e.String(http.StatusBadRequest, "missing username")
		}

		user, err := appStore.Users.GetUserByUsername(ctx, username)
		if err != nil || user == nil {
			return e.String(http.StatusNotFound, "user not found")
		}

		approvals, removals := authorTrust(ctx, appStore, user.ID)
		violations, _ := appStore.ModerationLog.ListViolationsByAuthor(ctx, user.ID)

		rows := make([]AdminUserViolationRow, 0, len(violations))
		for _, v := range violations {
			kind, label := violationKind(v.Action, v.NewState)
			rows = append(rows, AdminUserViolationRow{
				When:           v.CreatedAt,
				SchematicTitle: v.SchematicTitle,
				SchematicName:  v.SchematicName,
				Kind:           kind,
				KindLabel:      label,
				ActorType:      v.ActorType,
				ActorUsername:  v.ActorUsername,
				Reason:         v.Reason,
			})
		}

		d := AdminUserData{
			ProfileUsername: user.Username,
			ProfileID:       user.ID,
			ProfileEmail:    user.Email,
			ProfileAvatar:   user.Avatar,
			JoinedAt:        user.Created,
			IsBanned:        user.Deleted != nil,
			PublishedCount:  approvals,
			RemovedCount:    removals,
			ViolationCount:  len(rows),
			Violations:      rows,
		}
		d.Populate(e)
		d.AdminSection = "users"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Users"), "/admin/users", user.Username)
		d.Title = "User: " + user.Username
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminUserTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
