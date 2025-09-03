package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strconv"
)

var schematicsTemplates = append([]string{
	"./template/schematics.html",
	"./template/include/schematic_card.html",
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

func SchematicsHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
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
			"deleted = null && moderated = true",
			"-created",
			limit,
			offset)
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
		d.Title = "Create Mod Schematics"
		d.Categories = allCategories(app, cacheService)
		d.Description = "Find the latest Create Mod schematics listed here."
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
