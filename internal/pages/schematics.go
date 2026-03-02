package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strconv"
	"time"
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

func SchematicsHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"deleted = '' && moderated = true && (scheduled_at = null || scheduled_at <= {:now})",
			"-created",
			limit,
			offset,
			dbx.Params{"now": time.Now()},
		)
		if err != nil {
			return err
		}

		hasNext := len(results) > pageSize
		if hasNext {
			results = results[:pageSize]
		}

		d := SchematicsData{
			Schematics: MapResultsToSchematic(app, results, cacheService),
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
		d.Title = i18n.T(d.Language, "page.schematics.title")
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		d.Description = i18n.T(d.Language, "page.schematics.description")
		d.Slug = "/schematics"
		if len(d.Schematics) > 0 {
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematics[0].ID, d.Schematics[0].FeaturedImage)
		}

		html, err := registry.LoadFiles(schematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
