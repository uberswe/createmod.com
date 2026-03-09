package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

var adminDashboardTemplates = append([]string{
	"./template/admin_dashboard.html",
}, commonTemplates...)

type AdminDashboardData struct {
	DefaultData
	PendingCount     int64
	ModeratedCount   int64
	DeletedCount     int64
	TotalCount       int64
	ReportsCount     int
	PendingCatsCount int
	PendingTagsCount int
	RecentPending    []AdminSchematicItem
}

// AdminDashboardHandler renders the admin dashboard overview page at GET /admin.
func AdminDashboardHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		pendingCount, _ := appStore.Schematics.CountForAdmin(ctx, "pending")
		moderatedCount, _ := appStore.Schematics.CountForAdmin(ctx, "moderated")
		deletedCount, _ := appStore.Schematics.CountForAdmin(ctx, "deleted")
		totalCount, _ := appStore.Schematics.CountForAdmin(ctx, "")

		reports, _ := appStore.Reports.List(ctx, 1000, 0)
		pendingCats, _ := appStore.Categories.ListPending(ctx)
		pendingTags, _ := appStore.Tags.ListPending(ctx)

		recentPending, _ := appStore.Schematics.ListForAdmin(ctx, "pending", 10, 0)
		items := make([]AdminSchematicItem, 0, len(recentPending))
		for _, s := range recentPending {
			username := ""
			if s.AuthorID != "" {
				u, err := appStore.Users.GetUserByID(ctx, s.AuthorID)
				if err == nil && u != nil {
					username = u.Username
				}
			}
			items = append(items, AdminSchematicItem{
				ID:               s.ID,
				Title:            s.Title,
				Name:             s.Name,
				AuthorUsername:    username,
				Moderated:        s.Moderated,
				ModerationReason: s.ModerationReason,
				Blacklisted:      s.Blacklisted,
				Deleted:          s.Deleted != nil,
				Created:          s.Created,
				FeaturedImage:    s.FeaturedImage,
			})
		}

		d := AdminDashboardData{
			PendingCount:     pendingCount,
			ModeratedCount:   moderatedCount,
			DeletedCount:     deletedCount,
			TotalCount:       totalCount,
			ReportsCount:     len(reports),
			PendingCatsCount: len(pendingCats),
			PendingTagsCount: len(pendingTags),
			RecentPending:    items,
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Admin Dashboard")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminDashboardTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
