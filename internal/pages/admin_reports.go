package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"os"
	"time"
)

var adminReportsTemplates = append([]string{
	"./template/admin_reports.html",
}, commonTemplates...)

type AdminReportItem struct {
	ID         string
	TargetType string
	TargetID   string
	Reason     string
	Reporter   string
	Created    time.Time
}

type AdminReportsData struct {
	DefaultData
	Reports []AdminReportItem
}

func isSuperAdmin(e *core.RequestEvent, app *pocketbase.PocketBase) bool {
	if !isAuthenticated(e) {
		return false
	}
	super := os.Getenv("SUPERADMIN_EMAIL")
	if super == "" {
		// do not fallback to sender for access control; require explicit env to avoid accidental exposure
		return false
	}
	return authenticatedUserEmail(e) == super
}

// AdminReportsHandler renders a simple admin page listing recent reports.
func AdminReportsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !isSuperAdmin(e, app) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		coll, err := app.FindCollectionByNameOrId("reports")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "reports collection not available")
		}
		records, err := app.FindRecordsByFilter(coll.Id, "1=1", "-created", 100, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list reports")
		}
		items := make([]AdminReportItem, 0, len(records))
		for _, r := range records {
			items = append(items, AdminReportItem{
				ID:         r.Id,
				TargetType: r.GetString("target_type"),
				TargetID:   r.GetString("target_id"),
				Reason:     r.GetString("reason"),
				Reporter:   r.GetString("reporter"),
				Created:    r.GetDateTime("created").Time(),
			})
		}
		d := AdminReportsData{Reports: items}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Admin: Reports")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(adminReportsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminReportResolveHandler deletes a report by id (acts as resolve) and redirects back to /admin/reports.
func AdminReportResolveHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !isSuperAdmin(e, app) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		coll, err := app.FindCollectionByNameOrId("reports")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "reports collection not available")
		}
		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			// treat as already resolved
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/reports"))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/reports"))
		}
		if err := app.Delete(rec); err != nil {
			// best-effort; show error
			return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to resolve: %v", err))
		}
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/reports"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/reports"))
	}
}
