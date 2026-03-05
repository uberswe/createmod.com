package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
)

var adminTagsTemplates = append([]string{
	"./template/admin_tags.html",
}, commonTemplates...)

type AdminPendingCategory struct {
	ID   string
	Key  string
	Name string
}

type AdminPendingTag struct {
	ID   string
	Key  string
	Name string
}

type AdminTagsData struct {
	DefaultData
	PendingCategories []AdminPendingCategory
	PendingTags       []AdminPendingTag
}

// AdminTagsHandler renders the admin page for managing pending categories and tags.
func AdminTagsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		pendingCats, err := appStore.Categories.ListPending(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list pending categories")
		}
		pendingTags, err := appStore.Tags.ListPending(ctx)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list pending tags")
		}

		cats := make([]AdminPendingCategory, len(pendingCats))
		for i, c := range pendingCats {
			cats[i] = AdminPendingCategory{ID: c.ID, Key: c.Key, Name: c.Name}
		}
		tags := make([]AdminPendingTag, len(pendingTags))
		for i, t := range pendingTags {
			tags[i] = AdminPendingTag{ID: t.ID, Key: t.Key, Name: t.Name}
		}

		d := AdminTagsData{
			PendingCategories: cats,
			PendingTags:       tags,
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Admin: Tags & Categories")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminTagsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// AdminTagApproveHandler approves a pending category or tag by ID.
func AdminTagApproveHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		itemType := e.Request.URL.Query().Get("type")
		ctx := context.Background()

		switch itemType {
		case "category":
			if err := appStore.Categories.Approve(ctx, id); err != nil {
				return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to approve category: %v", err))
			}
			cacheService.Delete(cache.AllCategoriesKey)
		case "tag":
			if err := appStore.Tags.Approve(ctx, id); err != nil {
				return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to approve tag: %v", err))
			}
			cacheService.Delete(cache.AllTagsWithCountKey)
		default:
			return e.String(http.StatusBadRequest, "type must be 'category' or 'tag'")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/tags"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/tags"))
	}
}

// AdminTagRejectHandler deletes a pending category or tag by ID.
func AdminTagRejectHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		itemType := e.Request.URL.Query().Get("type")
		ctx := context.Background()

		switch itemType {
		case "category":
			if err := appStore.Categories.Delete(ctx, id); err != nil {
				return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete category: %v", err))
			}
			cacheService.Delete(cache.AllCategoriesKey)
		case "tag":
			if err := appStore.Tags.Delete(ctx, id); err != nil {
				return e.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete tag: %v", err))
			}
			cacheService.Delete(cache.AllTagsWithCountKey)
		default:
			return e.String(http.StatusBadRequest, "type must be 'category' or 'tag'")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/admin/tags"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/admin/tags"))
	}
}
