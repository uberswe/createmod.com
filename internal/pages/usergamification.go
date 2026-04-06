package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"createmod/internal/server"
)

var userPointsTemplates = append([]string{
	"./template/user-points.html",
}, commonTemplates...)

// PointLogEntry represents a single earned-points record for the template.
type PointLogEntry struct {
	Points      int
	Reason      string
	Description string
	EarnedAt    time.Time
}

// HowToEarnItem describes one way to earn points.
type HowToEarnItem struct {
	Action string
	Points int
}

// UserPointsData is the template data for /settings/points.
type UserPointsData struct {
	DefaultData
	Points       int
	PointLog     []PointLogEntry
	HowToEarn    []HowToEarnItem
	Page         int
	TotalPages   int
	PageSize     int
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
}

func UserPointsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)

		d := UserPointsData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Points"))
		d.Title = i18n.T(d.Language, "Points")
		d.Description = i18n.T(d.Language, "page.usergamification.description")
		d.Slug = "/settings/points"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Load user points
		ctx := e.Request.Context()
		if user, err := appStore.Users.GetUserByID(ctx, userID); err == nil && user != nil {
			d.Points = user.Points
		}

		// How to earn table
		d.HowToEarn = []HowToEarnItem{
			{Action: "Upload your first schematic", Points: 50},
			{Action: "First upload bonus", Points: 30},
			{Action: "Post your first comment", Points: 10},
		}

		// Pagination
		d.PageSize = 20
		d.Page = 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
				d.Page = pv
			}
		}

		offset := (d.Page - 1) * d.PageSize

		allEntries, _ := appStore.Achievements.GetPointLog(ctx, userID)
		totalCount := len(allEntries)

		d.TotalPages = (totalCount + d.PageSize - 1) / d.PageSize
		if d.TotalPages < 1 {
			d.TotalPages = 1
		}
		if d.Page > d.TotalPages {
			d.Page = d.TotalPages
		}

		// Slice to page
		end := offset + d.PageSize
		if end > totalCount {
			end = totalCount
		}
		if offset < totalCount {
			pageEntries := allEntries[offset:end]
			entries := make([]PointLogEntry, 0, len(pageEntries))
			for _, pe := range pageEntries {
				entries = append(entries, PointLogEntry{
					Points:      pe.Points,
					Reason:      pe.Reason,
					Description: pe.Description,
					EarnedAt:    pe.EarnedAt,
				})
			}
			d.PointLog = entries
		}

		d.HasPrev = d.Page > 1
		d.HasNext = d.Page < d.TotalPages
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/settings/points?p=%d", d.Page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/settings/points?p=%d", d.Page+1)
		}

		html, err := registry.LoadFiles(userPointsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
