package pages

import (
	"context"
	"createmod/internal/blockedurls"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var adminBlockedURLsTemplates = append([]string{
	"./template/admin_blocked_urls.html",
}, commonTemplates...)

// AdminBlockedURLRow is one blocked URL shown in the admin list.
type AdminBlockedURLRow struct {
	ID      string
	URL     string
	Note    string
	Created string
}

type AdminBlockedURLsData struct {
	DefaultData
	BlockedURLs []AdminBlockedURLRow
	FormError   string
}

// AdminBlockedURLsHandler renders the admin page for managing blocked URLs
// (e.g. DMCA takedown targets that must return a 404).
func AdminBlockedURLsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		list, err := appStore.BlockedURLs.List(context.Background())
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list blocked URLs")
		}
		rows := make([]AdminBlockedURLRow, len(list))
		for i, b := range list {
			rows[i] = AdminBlockedURLRow{
				ID:      b.ID,
				URL:     b.URL,
				Note:    b.Note,
				Created: b.Created.Format("2006-01-02 15:04"),
			}
		}

		d := AdminBlockedURLsData{
			BlockedURLs: rows,
			FormError:   e.Request.URL.Query().Get("error"),
		}
		d.Populate(e)
		d.AdminSection = "blocked-urls"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Blocked URLs"))
		d.Title = i18n.T(d.Language, "Admin: Blocked URLs")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminBlockedURLsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminBlockedURLCreateHandler adds a blocked URL. The submitted value may be
// a full URL or an absolute path; it is normalized to path + query.
func AdminBlockedURLCreateHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		normalized, err := blockedurls.Normalize(e.Request.FormValue("url"))
		if err != nil {
			return adminBlockedURLsRedirect(e, err.Error())
		}
		if strings.HasPrefix(normalized, "/admin") {
			return adminBlockedURLsRedirect(e, "refusing to block admin pages")
		}

		b := &store.BlockedURL{
			URL:       normalized,
			Note:      strings.TrimSpace(e.Request.FormValue("note")),
			CreatedBy: authenticatedUserID(e),
		}
		if err := appStore.BlockedURLs.Create(context.Background(), b); err != nil {
			return e.String(http.StatusInternalServerError, "failed to create blocked URL")
		}
		invalidateBlockedURLsCache(cacheService)
		return adminBlockedURLsRedirect(e, "")
	}
}

// AdminBlockedURLDeleteHandler removes a blocked URL so it is served normally
// again.
func AdminBlockedURLDeleteHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		if err := appStore.BlockedURLs.Delete(context.Background(), id); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete blocked URL")
		}
		invalidateBlockedURLsCache(cacheService)
		return adminBlockedURLsRedirect(e, "")
	}
}

// invalidateBlockedURLsCache clears the cached blocklist so changes take
// effect immediately on this pod and, via Redis pub/sub, on all other pods.
func invalidateBlockedURLsCache(cacheService *cache.Service) {
	if cacheService != nil {
		cacheService.Delete(cache.BlockedURLsKey)
	}
}

func adminBlockedURLsRedirect(e *server.RequestEvent, formError string) error {
	target := "/admin/blocked-urls"
	if formError != "" {
		target = fmt.Sprintf("%s?error=%s", target, url.QueryEscape(formError))
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, target))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, target))
}
