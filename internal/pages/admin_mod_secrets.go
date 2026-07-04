package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

var adminModSecretsTemplates = append([]string{
	"./template/admin_api_secrets.html",
}, commonTemplates...)

// AdminModSecretRow is one secret shown in the admin list.
type AdminModSecretRow struct {
	ID      string
	Label   string
	Note    string
	Secret  string
	Active  bool
	Created string
}

type AdminModSecretsData struct {
	DefaultData
	Secrets []AdminModSecretRow
}

// AdminModSecretsHandler renders the admin page for managing shared HMAC secrets.
func AdminModSecretsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		ctx := context.Background()
		list, err := appStore.ModSecrets.List(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list secrets")
		}
		rows := make([]AdminModSecretRow, len(list))
		for i, s := range list {
			rows[i] = AdminModSecretRow{
				ID:      s.ID,
				Label:   s.Label,
				Note:    s.Note,
				Secret:  s.Secret,
				Active:  s.Active,
				Created: s.Created.Format("2006-01-02 15:04"),
			}
		}

		d := AdminModSecretsData{Secrets: rows}
		d.Populate(e)
		d.AdminSection = "api-secrets"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "API Secrets"))
		d.Title = i18n.T(d.Language, "Admin: API Secrets")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminModSecretsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminModSecretCreateHandler adds a new secret. If the secret field is blank a
// random 64-hex value is generated server-side.
func AdminModSecretCreateHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		label := strings.TrimSpace(e.Request.FormValue("label"))
		note := strings.TrimSpace(e.Request.FormValue("note"))
		secret := strings.TrimSpace(e.Request.FormValue("secret"))
		if secret == "" {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				return e.String(http.StatusInternalServerError, "failed to generate secret")
			}
			secret = hex.EncodeToString(buf)
		}

		ms := &store.ModSecret{
			Label:     label,
			Note:      note,
			Secret:    secret,
			Active:    true,
			CreatedBy: authenticatedUserID(e),
		}
		if err := appStore.ModSecrets.Create(context.Background(), ms); err != nil {
			return e.String(http.StatusInternalServerError, "failed to create secret")
		}
		invalidateModSecretsCache(cacheService)
		return adminModSecretsRedirect(e)
	}
}

// AdminModSecretActiveHandler activates/deactivates a secret (?active=true|false).
func AdminModSecretActiveHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		active := e.Request.URL.Query().Get("active") == "true"
		if err := appStore.ModSecrets.SetActive(context.Background(), id, active); err != nil {
			return e.String(http.StatusInternalServerError, "failed to update secret")
		}
		invalidateModSecretsCache(cacheService)
		return adminModSecretsRedirect(e)
	}
}

// AdminModSecretDeleteHandler permanently removes a secret.
func AdminModSecretDeleteHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		if err := appStore.ModSecrets.Delete(context.Background(), id); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete secret")
		}
		invalidateModSecretsCache(cacheService)
		return adminModSecretsRedirect(e)
	}
}

// invalidateModSecretsCache clears the cached active-secret list so admin
// changes take effect on the next request (on this pod; other pods refresh
// within the cache TTL).
func invalidateModSecretsCache(cacheService *cache.Service) {
	if cacheService != nil {
		cacheService.Delete(modSecretsCacheKey)
	}
}

func adminModSecretsRedirect(e *server.RequestEvent) error {
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/api-secrets"))
		return e.HTML(http.StatusNoContent, "")
	}
	return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/api-secrets"))
}
