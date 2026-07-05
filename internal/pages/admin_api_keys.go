package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var adminAPIKeysTemplates = append([]string{
	"./template/admin_api_keys.html",
}, commonTemplates...)

// defaultAPIRateLimitPerMinute is the per-key limit applied when no
// admin-assigned override is set.
const defaultAPIRateLimitPerMinute = 120

// maxAPIRateLimitPerMinute caps admin-assigned overrides.
const maxAPIRateLimitPerMinute = 1_000_000

// AdminAPIKeyEndpointRow is one endpoint's usage for a key (last 30 days).
type AdminAPIKeyEndpointRow struct {
	Endpoint string
	Requests int64
	LastUsed string
}

// AdminAPIKeyRow is one API key shown in the admin list. Last8 is the only
// key material exposed — the same display fragment users see on their own
// settings page; never show more of the key.
type AdminAPIKeyRow struct {
	ID         string
	Username   string
	UserID     string
	Label      string
	Last8      string
	Created    string
	LastUsed   string // empty when the key has never been used
	Usage24h   int64
	Usage7d    int64
	UsageTotal int64
	RateLimit  int // 0 = endpoint default
	Endpoints  []AdminAPIKeyEndpointRow
}

type AdminAPIKeysData struct {
	DefaultData
	Keys             []AdminAPIKeyRow
	DefaultRateLimit int
	Query            string
}

// filterAdminAPIKeyRows returns the rows whose username or label contains q
// (case-insensitive). An empty q returns rows unchanged.
func filterAdminAPIKeyRows(rows []AdminAPIKeyRow, q string) []AdminAPIKeyRow {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return rows
	}
	filtered := make([]AdminAPIKeyRow, 0, len(rows))
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.Username), q) || strings.Contains(strings.ToLower(r.Label), q) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// AdminAPIKeysHandler renders the admin overview of all user API keys with
// usage aggregates and per-key rate limit overrides.
func AdminAPIKeysHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		ctx := context.Background()
		keys, err := appStore.APIKeys.ListAllWithUsage(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list API keys")
		}
		endpointUsage, err := appStore.APIKeys.UsageByEndpoint(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to load API key usage")
		}
		byKey := make(map[string][]AdminAPIKeyEndpointRow)
		for _, u := range endpointUsage {
			byKey[u.APIKeyID] = append(byKey[u.APIKeyID], AdminAPIKeyEndpointRow{
				Endpoint: u.Endpoint,
				Requests: u.Requests,
				LastUsed: u.LastUsed.Format("2006-01-02 15:04"),
			})
		}

		rows := make([]AdminAPIKeyRow, len(keys))
		for i, k := range keys {
			lastUsed := ""
			if !k.LastUsed.IsZero() {
				lastUsed = k.LastUsed.Format("2006-01-02 15:04")
			}
			rows[i] = AdminAPIKeyRow{
				ID:         k.ID,
				Username:   k.Username,
				UserID:     k.UserID,
				Label:      k.Label,
				Last8:      k.Last8,
				Created:    k.Created.Format("2006-01-02 15:04"),
				LastUsed:   lastUsed,
				Usage24h:   k.Usage24h,
				Usage7d:    k.Usage7d,
				UsageTotal: k.UsageTotal,
				RateLimit:  k.RateLimitPerMinute,
				Endpoints:  byKey[k.ID],
			}
		}

		query := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		rows = filterAdminAPIKeyRows(rows, query)

		d := AdminAPIKeysData{Keys: rows, DefaultRateLimit: defaultAPIRateLimitPerMinute, Query: query}
		d.Populate(e)
		d.AdminSection = "api-keys"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "API Keys"))
		d.Title = i18n.T(d.Language, "Admin: API Keys")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminAPIKeysTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminAPIKeyRateLimitHandler sets or clears the per-minute rate limit
// override for one key. An empty or zero value clears the override so the
// endpoint defaults apply again.
func AdminAPIKeyRateLimitHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		raw := strings.TrimSpace(e.Request.FormValue("rate_limit"))
		limit := 0
		if raw != "" {
			v, err := strconv.Atoi(raw)
			if err != nil || v < 0 || v > maxAPIRateLimitPerMinute {
				return e.String(http.StatusBadRequest, "rate limit must be a number between 0 and 1000000")
			}
			limit = v
		}
		if err := appStore.APIKeys.SetRateLimit(context.Background(), id, limit); err != nil {
			return e.String(http.StatusInternalServerError, "failed to update rate limit")
		}
		return adminAPIKeysRedirect(e)
	}
}

func adminAPIKeysRedirect(e *server.RequestEvent) error {
	// Preserve the active search filter across the redirect.
	target := "/admin/api-keys"
	if q := strings.TrimSpace(e.Request.FormValue("q")); q != "" {
		target += "?q=" + url.QueryEscape(q)
	}
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, target))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, target))
}
