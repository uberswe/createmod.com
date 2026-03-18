package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strings"
	"time"
)

var adminModsTemplates = append([]string{
	"./template/admin_mods.html",
}, commonTemplates...)

var adminModEditTemplates = append([]string{
	"./template/admin_mod_edit.html",
}, commonTemplates...)

type AdminModItem struct {
	Namespace     string
	DisplayName   string
	CurseforgeURL string
	ModrinthURL   string
	IconURL       string
	ManuallySet   bool
	LastFetched   *time.Time
}

type AdminModsData struct {
	DefaultData
	Mods []AdminModItem
}

type AdminModEditData struct {
	DefaultData
	Mod store.ModMetadata
}

// AdminModsHandler renders the admin page listing all mod metadata.
func AdminModsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()
		all, err := appStore.ModMetadata.ListAll(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list mod metadata")
		}

		items := make([]AdminModItem, len(all))
		for i, m := range all {
			items[i] = AdminModItem{
				Namespace:     m.Namespace,
				DisplayName:   m.DisplayName,
				CurseforgeURL: m.CurseforgeURL,
				ModrinthURL:   m.ModrinthURL,
				IconURL:       m.IconURL,
				ManuallySet:   m.ManuallySet,
				LastFetched:   m.LastFetched,
			}
		}

		d := AdminModsData{Mods: items}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Mods"))
		d.Title = i18n.T(d.Language, "Admin: Mods")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminModsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminModEditHandler renders the edit form for a single mod's metadata.
func AdminModEditHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		namespace := e.Request.PathValue("namespace")
		if namespace == "" {
			return e.String(http.StatusBadRequest, "missing namespace")
		}

		ctx := context.Background()
		meta, err := appStore.ModMetadata.GetByNamespace(ctx, namespace)
		if err != nil || meta == nil {
			return e.String(http.StatusNotFound, "mod not found")
		}

		d := AdminModEditData{Mod: *meta}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Mods"), "/admin/mods", i18n.T(d.Language, "Edit"))
		d.Title = i18n.T(d.Language, "Edit Mod:") + " " + namespace
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminModEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminModUpdateHandler processes the edit form submission for a mod's metadata.
func AdminModUpdateHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		namespace := e.Request.PathValue("namespace")
		if namespace == "" {
			return e.String(http.StatusBadRequest, "missing namespace")
		}

		ctx := context.Background()
		existing, err := appStore.ModMetadata.GetByNamespace(ctx, namespace)
		if err != nil || existing == nil {
			return e.String(http.StatusNotFound, "mod not found")
		}

		_ = e.Request.ParseForm()

		existing.DisplayName = strings.TrimSpace(e.Request.FormValue("display_name"))
		existing.Description = strings.TrimSpace(e.Request.FormValue("description"))
		existing.IconURL = strings.TrimSpace(e.Request.FormValue("icon_url"))
		existing.CurseforgeURL = strings.TrimSpace(e.Request.FormValue("curseforge_url"))
		existing.CurseforgeID = strings.TrimSpace(e.Request.FormValue("curseforge_id"))
		existing.ModrinthURL = strings.TrimSpace(e.Request.FormValue("modrinth_url"))
		existing.ModrinthSlug = strings.TrimSpace(e.Request.FormValue("modrinth_slug"))
		existing.SourceURL = strings.TrimSpace(e.Request.FormValue("source_url"))
		existing.ManuallySet = e.Request.FormValue("manually_set") == "on"

		if err := appStore.ModMetadata.Upsert(ctx, existing); err != nil {
			return e.String(http.StatusInternalServerError, "failed to update mod metadata")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/mods"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/mods"))
	}
}
