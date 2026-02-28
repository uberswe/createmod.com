package pages

import (
	"createmod/internal/cache"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
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

func UserPointsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserPointsData{}
		d.Populate(e)
		d.Title = "Points"
		d.Description = "Your points and earning history."
		d.Slug = "/settings/points"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		// Load user points
		if urec, err := app.FindRecordById("_pb_users_auth_", e.Auth.Id); err == nil && urec != nil {
			d.Points = urec.GetInt("points")
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

		// Query point_log for this user
		offset := (d.Page - 1) * d.PageSize
		allRecs, _ := app.FindRecordsByFilter("point_log", "user = {:u}", "-earned_at", -1, 0, dbx.Params{"u": e.Auth.Id})
		totalCount := len(allRecs)

		d.TotalPages = (totalCount + d.PageSize - 1) / d.PageSize
		if d.TotalPages < 1 {
			d.TotalPages = 1
		}
		if d.Page > d.TotalPages {
			d.Page = d.TotalPages
		}

		// Get the page slice
		recs, _ := app.FindRecordsByFilter("point_log", "user = {:u}", "-earned_at", d.PageSize, offset, dbx.Params{"u": e.Auth.Id})

		entries := make([]PointLogEntry, 0, len(recs))
		for _, r := range recs {
			entries = append(entries, PointLogEntry{
				Points:      r.GetInt("points"),
				Reason:      r.GetString("reason"),
				Description: r.GetString("description"),
				EarnedAt:    r.GetDateTime("earned_at").Time(),
			})
		}
		d.PointLog = entries

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
