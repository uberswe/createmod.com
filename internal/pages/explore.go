package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"math/rand/v2"
	"net/http"
)

var exploreTemplates = []string{
	"./template/dist/explore.html",
}

type ExploreData struct {
	DefaultData
	Images []models.ImageData
}

func ExploreHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		var results []core.Record
		err = app.RecordQuery(schematicsCollection).Select("id", "name", "title", "featured_image", "gallery").All(&results)
		if err != nil {
			return err
		}

		images := make([]models.ImageData, 0)
		for _, result := range results {
			for _, g := range result.GetStringSlice("gallery") {
				images = append(images, models.ImageData{
					ID:    result.Id,
					Title: result.GetString("title"),
					Name:  result.GetString("name"),
					Image: g,
				})
			}
			images = append(images, models.ImageData{
				ID:    result.Id,
				Title: result.GetString("title"),
				Name:  result.GetString("name"),
				Image: result.GetString("featured_image"),
			})
		}

		show := len(images)
		if show > 1000 {
			show = 1000
		}
		dest := make([]models.ImageData, show)
		perm := rand.Perm(show)
		for i, v := range perm {
			dest[v] = images[i]
		}

		d := ExploreData{
			Images: dest,
		}
		d.Populate(e)
		d.Title = "Explore Create Mod Schematics"
		d.Categories = allCategories(app, cacheService)
		d.Description = "Explore a random gallery of Create Mod schematics"
		d.Slug = "/explore"
		if len(dest) > 0 {
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", dest[0].ID, dest[0].Image)
		}
		html, err := registry.LoadFiles(exploreTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
