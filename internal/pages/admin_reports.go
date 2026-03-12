package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

var adminReportsTemplates = append([]string{
	"./template/admin_reports.html",
}, commonTemplates...)

type AdminReportItem struct {
	ID             string
	TargetType     string
	TargetID       string
	TargetURL      string // clickable link to the reported item
	TargetLabel    string // human-readable label for the link
	Reason         string
	Reporter       string
	ReporterName   string // username if the reporter was logged in
	Created        time.Time
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
			item := AdminReportItem{
				ID:         r.ID,
				TargetType: r.TargetType,
				TargetID:   r.TargetID,
				Reason:     r.Reason,
				Reporter:   r.Reporter,
				Created:    r.Created,
			}
			item.TargetURL, item.TargetLabel = resolveReportTarget(appStore, r.TargetType, r.TargetID)
			if r.Reporter != "" {
				if u, err := appStore.Users.GetUserByID(context.Background(), r.Reporter); err == nil && u != nil {
					item.ReporterName = u.Username
				}
			}
			items = append(items, item)
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

// resolveReportTarget builds a URL and label for the reported item.
func resolveReportTarget(appStore *store.Store, targetType, targetID string) (string, string) {
	ctx := context.Background()
	switch targetType {
	case "schematic":
		s, err := appStore.Schematics.GetByIDAdmin(ctx, targetID)
		if err == nil && s != nil {
			return "/schematics/" + s.Name, s.Title
		}
		return "/admin/schematics/" + targetID, targetID
	case "comment":
		c, err := appStore.Comments.GetByID(ctx, targetID)
		if err == nil && c != nil && c.SchematicID != nil {
			s, sErr := appStore.Schematics.GetByIDAdmin(ctx, *c.SchematicID)
			if sErr == nil && s != nil {
				return "/schematics/" + s.Name + "#comment-" + targetID, truncate(c.Content, 60)
			}
		}
		return "", targetID
	case "collection":
		coll, err := appStore.Collections.GetByID(ctx, targetID)
		if err == nil && coll != nil {
			slug := coll.Slug
			if slug == "" {
				slug = coll.ID
			}
			title := coll.Title
			if title == "" {
				title = coll.Name
			}
			return "/collections/" + slug, title
		}
		return "/collections/" + targetID, targetID
	}
	return "", targetID
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// AdminReportResolveHandler resolves a report: deletes it, optionally emails the reporter.
func AdminReportResolveHandler(appStore *store.Store, mailService *mailer.Service) func(e *server.RequestEvent) error {
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

		_ = e.Request.ParseForm()
		resolutionNote := strings.TrimSpace(e.Request.FormValue("resolution_note"))
		notifyReporter := e.Request.FormValue("notify_reporter") == "true"

		// Look up the report before deleting so we can email the reporter
		ctx := context.Background()
		var reporterEmail string
		if notifyReporter && mailService != nil {
			reports, _ := appStore.Reports.List(ctx, 100, 0)
			for _, r := range reports {
				if r.ID == id && r.Reporter != "" {
					if u, err := appStore.Users.GetUserByID(ctx, r.Reporter); err == nil && u != nil && u.Email != "" {
						reporterEmail = u.Email
					}
					break
				}
			}
		}

		if err := appStore.Reports.Delete(ctx, id); err != nil {
			return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to resolve: %v", err))
		}

		// Send resolution email to reporter
		if reporterEmail != "" && mailService != nil {
			go func() {
				subject := "Your report has been reviewed"
				body := "Your report has been reviewed by an administrator."
				if resolutionNote != "" {
					body += "\n\nResolution note: " + resolutionNote
				}
				htmlBody := mailer.SchematicEmailHTML("Report Resolution", "", "", body)
				msg := &mailer.Message{
					From:    mailService.DefaultFrom(),
					To:      []mail.Address{{Address: reporterEmail}},
					Subject: subject,
					HTML:    htmlBody,
				}
				if err := mailService.Send(msg); err != nil {
					slog.Error("failed to send report resolution email", "error", err, "to", reporterEmail)
				}
			}()
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/reports"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/reports"))
	}
}

// AdminReportDeleteTargetHandler deletes the reported item and all related reports.
func AdminReportDeleteTargetHandler(appStore *store.Store) func(e *server.RequestEvent) error {
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

		ctx := context.Background()
		// Look up the report to find the target
		reports, err := appStore.Reports.List(ctx, 100, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to look up report")
		}
		var target *store.Report
		for _, r := range reports {
			if r.ID == id {
				target = &r
				break
			}
		}
		if target == nil {
			return e.String(http.StatusNotFound, "report not found")
		}

		// Delete the target
		switch target.TargetType {
		case "schematic":
			if err := appStore.Schematics.SoftDelete(ctx, target.TargetID); err != nil {
				slog.Error("admin report: failed to delete schematic", "error", err, "id", target.TargetID)
				return e.String(http.StatusInternalServerError, "failed to delete schematic")
			}
		case "comment":
			if err := appStore.Comments.Delete(ctx, target.TargetID); err != nil {
				slog.Error("admin report: failed to delete comment", "error", err, "id", target.TargetID)
				return e.String(http.StatusInternalServerError, "failed to delete comment")
			}
		case "collection":
			if err := appStore.Collections.SoftDelete(ctx, target.TargetID); err != nil {
				slog.Error("admin report: failed to delete collection", "error", err, "id", target.TargetID)
				return e.String(http.StatusInternalServerError, "failed to delete collection")
			}
		default:
			return e.String(http.StatusBadRequest, "unknown target type: "+target.TargetType)
		}

		// Remove all reports for this target
		_, _ = appStore.Reports.DeleteByTarget(ctx, target.TargetType, target.TargetID)

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/reports"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/reports"))
	}
}

// AdminReportIgnoreHandler removes all reports for this target (dismiss without action).
func AdminReportIgnoreHandler(appStore *store.Store) func(e *server.RequestEvent) error {
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

		ctx := context.Background()
		// Look up the report to find the target
		reports, err := appStore.Reports.List(ctx, 100, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to look up report")
		}
		var target *store.Report
		for _, r := range reports {
			if r.ID == id {
				target = &r
				break
			}
		}
		if target == nil {
			return e.String(http.StatusNotFound, "report not found")
		}

		// Remove all reports for this target
		_, _ = appStore.Reports.DeleteByTarget(ctx, target.TargetType, target.TargetID)

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/reports"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/reports"))
	}
}
