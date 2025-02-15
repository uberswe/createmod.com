package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var schematicsTemplates = []string{
	"./template/dist/schematics.html",
	"./template/dist/include/schematic_card.html",
}

type SchematicsData struct {
	DefaultData
	Schematics []models.Schematic
}

func SchematicsHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"1=1",
			"-created",
			51,
			0)

		d := SchematicsData{
			Schematics: MapResultsToSchematic(app, results, cacheService),
		}
		d.Populate(e)
		d.Title = "Create Mod Schematics"
		d.Categories = allCategories(app, cacheService)
		d.Description = "Find the latest Create Mod schematics listed here."
		d.Slug = "/schematics"
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematics[0].ID, d.Schematics[0].FeaturedImage)

		html, err := registry.LoadFiles(schematicsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
