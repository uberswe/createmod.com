package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"net/http"
	"strconv"
)

var mySchematicsTemplates = append([]string{
	"./template/my_schematics.html",
	"./template/include/schematic_card.html",
}, commonTemplates...)

type MySchematicsData struct {
	DefaultData
	Schematics []models.Schematic
	Page       int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

func MySchematicsHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		userID := authenticatedUserID(e)

		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		limit := pageSize + 1
		offset := (page - 1) * pageSize

		results, err := appStore.Schematics.ListByAuthorAll(context.Background(), userID, limit, offset)
		if err != nil {
			return err
		}

		hasNext := len(results) > pageSize
		if hasNext {
			results = results[:pageSize]
		}

		d := MySchematicsData{
			Schematics: MapStoreSchematics(appStore, results, cacheService),
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/my-schematics?p=%d", page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/my-schematics?p=%d", page+1)
		}

		d.Populate(e)
		translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "My Schematics"))
		d.Title = i18n.T(d.Language, "My Schematics")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/my-schematics"
		d.NoIndex = true

		html, err := registry.LoadFiles(mySchematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
