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

var adminCollectionsTemplates = append([]string{
	"./template/admin_collections.html",
}, commonTemplates...)

type AdminCollectionItem struct {
	ID        string
	Title     string
	Slug      string
	Author    string
	Published bool
	Deleted   bool
	Views     int
	Created   time.Time
	Updated   time.Time
}

type AdminCollectionsData struct {
	DefaultData
	Collections []AdminCollectionItem
	Filter      string
	Page        int
	TotalPages  int
	Total       int64
	PrevPage    int
	NextPage    int
}

const adminCollectionsPerPage = 20

func AdminCollectionsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		ctx := context.Background()

		filter := e.Request.URL.Query().Get("filter")
		if filter == "" {
			filter = "all"
		}
		if filter != "all" && filter != "published" && filter != "unpublished" && filter != "deleted" {
			filter = "all"
		}

		pageStr := e.Request.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * adminCollectionsPerPage

		collections, err := appStore.Collections.ListForAdmin(ctx, filter, adminCollectionsPerPage, offset)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to list collections")
		}

		total, err := appStore.Collections.CountForAdmin(ctx, filter)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to count collections")
		}

		totalPages := int(total) / adminCollectionsPerPage
		if int(total)%adminCollectionsPerPage != 0 {
			totalPages++
		}
		if totalPages < 1 {
			totalPages = 1
		}

		items := make([]AdminCollectionItem, 0, len(collections))
		for _, c := range collections {
			username := ""
			if c.AuthorID != nil && *c.AuthorID != "" {
				u, err := appStore.Users.GetUserByID(ctx, *c.AuthorID)
				if err == nil && u != nil {
					username = u.Username
				}
			}
			items = append(items, AdminCollectionItem{
				ID:        c.ID,
				Title:     c.Title,
				Slug:      c.Slug,
				Author:    username,
				Published: c.Published,
				Deleted:   c.Deleted != "",
				Views:     c.Views,
				Created:   c.Created,
				Updated:   c.Updated,
			})
		}

		d := AdminCollectionsData{
			Collections: items,
			Filter:      filter,
			Page:        page,
			TotalPages:  totalPages,
			Total:       total,
			PrevPage:    page - 1,
			NextPage:    page + 1,
		}
		d.Populate(e)
		d.AdminSection = "collections"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Admin"), "/admin", i18n.T(d.Language, "Collections"))
		d.Title = i18n.T(d.Language, "Admin: Collections")
		d.SubCategory = "Admin"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(adminCollectionsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func AdminCollectionDeleteHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		if err := appStore.Collections.SoftDelete(context.Background(), id); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete collection")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/admin/collections")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/admin/collections")
	}
}

func AdminCollectionUnpublishHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isSuperAdmin(e) {
			return e.String(http.StatusForbidden, "forbidden")
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}

		ctx := context.Background()
		coll, err := appStore.Collections.GetByID(ctx, id)
		if err != nil || coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}

		coll.Published = false
		if err := appStore.Collections.Update(ctx, coll); err != nil {
			return e.String(http.StatusInternalServerError, "failed to unpublish collection")
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/admin/collections")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/admin/collections")
	}
}
