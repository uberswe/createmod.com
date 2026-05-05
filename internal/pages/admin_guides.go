package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strconv"
	"time"
)

var adminGuidesTemplates = append([]string{
	"./template/admin_guides.html",
}, commonTemplates...)

type AdminGuideItem struct {
	ID      string
	Title   string
	Slug    string
	Author  string
	Views   int
	Created time.Time
	Updated time.Time
}

type AdminGuidesData struct {
	DefaultData
	Guides     []AdminGuideItem
	Filter     string
	Page       int
	TotalPages int
	Total      int64
	PrevPage   int
	NextPage   int
}

const adminGuidesPerPage = 20

func AdminGuidesHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		filter := e.Request.URL.Query().Get("filter")
		if filter == "" {
			filter = "active"
		}
		if filter != "all" && filter != "active" && filter != "deleted" {
			filter = "active"
		}

		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * adminGuidesPerPage

		guides, err := appStore.Guides.ListForAdmin(ctx, filter, adminGuidesPerPage, offset)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list guides")
		}

		total, err := appStore.Guides.CountForAdmin(ctx, filter)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to count guides")
		}

		totalPages := int(total) / adminGuidesPerPage
		if int(total)%adminGuidesPerPage != 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		items := make([]AdminGuideItem, 0, len(guides))
		for _, g := range guides {
			username := ""
			if g.AuthorID != nil && *g.AuthorID != "" {
				u, err := appStore.Users.GetUserByID(ctx, *g.AuthorID)
				if err == nil && u != nil {
					username = u.Username
				}
			}
			items = append(items, AdminGuideItem{
				ID:      g.ID,
				Title:   g.Title,
				Slug:    g.Slug,
				Author:  username,
				Views:   g.Views,
				Created: g.Created,
				Updated: g.Updated,
			})
		}

		d := AdminGuidesData{
			Guides:     items,
			Filter:     filter,
			Page:       page,
			TotalPages: totalPages,
			Total:      total,
			PrevPage:   page - 1,
			NextPage:   page + 1,
		}
		d.Populate(e)
		d.AdminSection = "guides"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Guides"))
		d.Title = i18n.T(d.Language, "Admin: Guides")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminGuidesTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func AdminGuideDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		if err := appStore.Guides.SoftDelete(context.Background(), id); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete guide")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/admin/guides")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/admin/guides")
	}
}
