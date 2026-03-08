package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/session"
	"createmod/internal/store"
	"fmt"
	"createmod/internal/server"
	"net/http"
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

func isSuperAdmin(e *server.RequestEvent) bool {
	user := session.UserFromContext(e.Request.Context())
	return user != nil && user.IsAdmin
}

// AdminReportsHandler renders a simple admin page listing recent reports.
func AdminReportsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		reports, err := appStore.Reports.List(context.Background(), 100, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list reports")
		}
		items := make([]AdminReportItem, 0, len(reports))
		for _, r := range reports {
			items = append(items, AdminReportItem{
				ID:         r.ID,
				TargetType: r.TargetType,
				TargetID:   r.TargetID,
				Reason:     r.Reason,
				Reporter:   r.Reporter,
				Created:    r.Created,
			})
		}
		d := AdminReportsData{Reports: items}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Admin: Reports")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(adminReportsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminReportResolveHandler deletes a report by id (acts as resolve) and redirects back to /admin/reports.
func AdminReportResolveHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		if err := appStore.Reports.Delete(context.Background(), id); err != nil {
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
