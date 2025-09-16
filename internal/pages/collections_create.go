package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const collectionsNewTemplate = "./template/collections_new.html"

var collectionsNewTemplates = append([]string{
	collectionsNewTemplate,
}, commonTemplates...)

type CollectionsNewData struct {
	DefaultData
	Error string
}

// CollectionsNewHandler renders the new collection form.
func CollectionsNewHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := CollectionsNewData{}
		d.Populate(e)
		d.Title = "Create collection"
		d.Description = "Create a new collection of schematics"
		d.Slug = "/collections/new"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(collectionsNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// CollectionsCreateHandler handles POST /collections to create a collection record in PB.
func CollectionsCreateHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			// Require login to create a collection
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		title := e.Request.FormValue("title")
		if title == "" {
			title = e.Request.FormValue("name")
		}
		description := e.Request.FormValue("description")
		bannerURL := e.Request.FormValue("banner_url")

		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		rec := core.NewRecord(coll)
		if title != "" {
			rec.Set("title", title)
			rec.Set("name", title)
		}
		if description != "" {
			rec.Set("description", description)
		}
		if bannerURL != "" {
			rec.Set("banner_url", bannerURL)
		}
		rec.Set("author", e.Auth.Id)
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}
		// After create, go back to listing (detail page may not exist yet)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", "/collections")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/collections")
	}
}
