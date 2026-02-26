package pages

import (
	"createmod/internal/cache"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"math/rand/v2"
	"net/http"
)

var exploreTemplates = append([]string{
	"./template/explore.html",
}, commonTemplates...)

type ExploreImage struct {
	ID    string
	Name  string
	Title string
	Image string
	Size  string // "sm", "md", or "lg"
}

type ExploreData struct {
	DefaultData
	Images []ExploreImage
}

func ExploreHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		var results []core.Record
		err = app.RecordQuery(schematicsCollection).
			Select("schematics.id", "schematics.name", "schematics.title", "schematics.featured_image", "schematics.gallery").
			Where(dbx.And(
				dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL)"),
				dbx.NewExp("schematics.moderated = 1"),
				dbx.NewExp("(schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))"),
				dbx.NewExp("schematics.featured_image != ''"),
				dbx.NewExp("schematics.name != ''"),
			)).
			OrderBy("RANDOM()").
			All(&results)
		if err != nil {
			return err
		}

		images := make([]ExploreImage, 0)
		for _, result := range results {
			for _, g := range result.GetStringSlice("gallery") {
				if g == "" {
					continue
				}
				images = append(images, ExploreImage{
					ID:    result.Id,
					Title: result.GetString("title"),
					Name:  result.GetString("name"),
					Image: g,
				})
			}
			if fi := result.GetString("featured_image"); fi != "" {
				images = append(images, ExploreImage{
					ID:    result.Id,
					Title: result.GetString("title"),
					Name:  result.GetString("name"),
					Image: fi,
				})
			}
		}

		show := len(images)
		if show > 1000 {
			show = 1000
		}
		dest := make([]ExploreImage, show)
		perm := rand.Perm(show)
		for i, v := range perm {
			dest[v] = images[i]
		}

		// Assign varied sizes for visual interest in the masonry grid.
		// Roughly 15% large, 35% medium, 50% small.
		for i := range dest {
			r := rand.IntN(100)
			switch {
			case r < 15:
				dest[i].Size = "lg"
			case r < 50:
				dest[i].Size = "md"
			default:
				dest[i].Size = "sm"
			}
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
