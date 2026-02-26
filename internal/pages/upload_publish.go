package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const uploadPublishTemplate = "./template/upload_publish.html"

var uploadPublishTemplates = append([]string{
	uploadPublishTemplate,
	uploadStepsTemplate,
}, commonTemplates...)

// UploadPublishHandler renders the publish form for a previously uploaded temp schematic.
// Requires authentication; redirects to /login if not logged in.
func UploadPublishHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusFound, "/login")
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		// Resolve entry from PB or in-memory store
		var entry tempUpload
		if pbEntry, ok := loadTempUploadPB(app, token); ok {
			entry = pbEntry
		} else {
			tempUploadStore.RLock()
			v, ok := tempUploadStore.m[token]
			tempUploadStore.RUnlock()
			if !ok {
				return e.String(http.StatusNotFound, "invalid or expired token")
			}
			entry = v
		}

		d := UploadPublishData{}
		d.Populate(e)
		d.UploadStep = 3
		d.Title = "Publish Schematic"
		d.Description = "Add details and publish your schematic to the community."
		d.Slug = "/u/" + token + "/publish"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
		d.Token = token
		d.Filename = entry.Filename
		d.Size = entry.Size
		d.BlockCount = entry.BlockCount
		d.DimX = entry.DimX
		d.DimY = entry.DimY
		d.DimZ = entry.DimZ
		d.Tags = allTags(app)
		d.MinecraftVersions = allMinecraftVersions(app)
		d.CreatemodVersions = allCreatemodVersions(app)
		d.AdditionalFiles = loadTempUploadFiles(app, token)

		html, err := registry.LoadFiles(uploadPublishTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
