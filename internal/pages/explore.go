package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"fmt"
	"createmod/internal/server"
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

func ExploreHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Fetch all approved schematics (we need gallery images, so fetch all)
		storeSchematics, err := appStore.Schematics.ListAllForIndex(context.Background())
		if err != nil {
			return err
		}

		images := make([]ExploreImage, 0)
		for _, s := range storeSchematics {
			if s.FeaturedImage == "" || s.Name == "" {
				continue
			}
			for _, g := range s.Gallery {
				if g == "" {
					continue
				}
				images = append(images, ExploreImage{
					ID:    s.ID,
					Title: s.Title,
					Name:  s.Name,
					Image: g,
				})
			}
			images = append(images, ExploreImage{
				ID:    s.ID,
				Title: s.Title,
				Name:  s.Name,
				Image: s.FeaturedImage,
			})
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
		d.Title = i18n.T(d.Language, "page.explore.title")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Description = i18n.T(d.Language, "page.explore.description")
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
