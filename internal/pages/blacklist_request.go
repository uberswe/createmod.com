package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var blacklistRequestTemplates = append([]string{
	"./template/blacklist_request.html",
}, commonTemplates...)

type BlacklistRequestData struct {
	DefaultData
}

// BlacklistRequestHandler renders a simple page that lets authenticated users
// submit a blacklist request using the existing /reports endpoint.
func BlacklistRequestHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require login so we can associate reporter + user id
		if e.Auth == nil {
			return e.Redirect(http.StatusFound, "/login")
		}
		d := BlacklistRequestData{}
		d.Populate(e)
		d.Title = "Request schematic blacklisting"
		d.Description = "Submit a request to blacklist a schematic you own."
		d.Slug = "/blacklist-request"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(blacklistRequestTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
