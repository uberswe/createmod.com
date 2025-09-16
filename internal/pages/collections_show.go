package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var collectionsShowTemplates = append([]string{
	"./template/collections_show.html",
}, commonTemplates...)

// CollectionsShowData represents data for a single collection view page.
type CollectionsShowData struct {
	DefaultData
	TitleText   string
	Description string
	BannerURL   string
	Views       int
	Featured    bool
	IsOwner     bool
}

// CollectionsShowHandler renders a basic collection detail page by slug or id.
// It degrades gracefully if the PocketBase collection is not available yet.
func CollectionsShowHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")

		d := CollectionsShowData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)
		d.Slug = "/collections/" + slug

		if coll, err := app.FindCollectionByNameOrId("collections"); err == nil && coll != nil {
			// Try to find by slug first, fallback to id
			var rec *core.Record
			// by slug
			if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
				rec = r[0]
			}
			if rec == nil {
				if r, err := app.FindRecordById(coll.Id, slug); err == nil {
					rec = r
				}
			}
			if rec != nil {
				d.TitleText = rec.GetString("title")
				if d.TitleText == "" {
					d.TitleText = rec.GetString("name")
				}
				d.Description = rec.GetString("description")
				d.BannerURL = rec.GetString("banner_url")
				d.Featured = rec.GetBool("featured")
				if e.Auth != nil && rec.GetString("author") == e.Auth.Id {
					d.IsOwner = true
				}
				// Best-effort views increment
				currentViews := rec.GetInt("views")
				rec.Set("views", currentViews+1)
				if err := app.Save(rec); err == nil {
					d.Views = currentViews + 1
				} else {
					// If the schema doesn't have the field or save fails, continue without blocking
					d.Views = currentViews
					app.Logger().Warn("collections: failed to increment views", "error", err)
				}
				// SEO/meta
				d.Title = d.TitleText
				if d.Title == "" {
					d.Title = "Collection"
				}
				if d.Description == "" {
					d.Description = "Collection details"
				}
			} else {
				// Not found
				d.Title = "Collection not found"
				d.Description = "We couldn't find this collection."
			}
		} else {
			// No PB schema available yet
			d.Title = "Collection"
			d.Description = "Collection details"
		}

		html, err := registry.LoadFiles(collectionsShowTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
