package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"createmod/internal/server"
	"net/http"
	"net/url"
	"strconv"
)

var schematicsTemplates = append([]string{
	"./template/schematics.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

type SchematicsData struct {
	DefaultData
	Schematics []models.Schematic
	Page       int
	PageSize   int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

func SchematicsHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Pagination params
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		limit := pageSize + 1 // fetch one extra to detect next page
		offset := (page - 1) * pageSize

		results, err := appStore.Schematics.ListApproved(context.Background(), limit, offset)
		if err != nil {
			return err
		}

		hasNext := len(results) > pageSize
		if hasNext {
			results = results[:pageSize]
		}

		d := SchematicsData{
			Schematics: MapStoreSchematics(appStore, results, cacheService),
			Page:       page,
			PageSize:   pageSize,
			HasPrev:    page > 1,
			HasNext:    hasNext,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/schematics?p=%d", page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/schematics?p=%d", page+1)
		}

		d.Populate(e)
		translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Schematics"))
		d.Title = i18n.T(d.Language, "page.schematics.title")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Description = i18n.T(d.Language, "page.schematics.description")
		d.Slug = "/schematics"
		if len(d.Schematics) > 0 {
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematics[0].ID, url.PathEscape(d.Schematics[0].FeaturedImage))
		}

		html, err := registry.LoadFiles(schematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
