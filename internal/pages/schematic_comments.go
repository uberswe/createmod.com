package pages

import (
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/i18n"
	"createmod/internal/search"
	"createmod/internal/store"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	template2 "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var schematicCommentsTemplates = []string{
	"./template/include/comments.html",
}

// SchematicCommentsHandler returns only the comments list for a schematic.
// Useful for HTMX partial refresh of comments.
func SchematicCommentsHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template2.Registry, _ *discord.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name} && deleted = ''",
			"-created",
			1,
			0,
			dbx.Params{"name": e.Request.PathValue("name")})

		if len(results) != 1 {
			nd := DefaultData{}
			nd.Populate(e)
			nd.Title = i18n.T(nd.Language, "Page Not Found")
			html, err := registry.LoadFiles(fourOhFourTemplates...).Render(nd)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d := SchematicData{
			Schematic: mapResultToSchematic(app, results[0], cacheService),
		}
		d.Populate(e)
		d.Comments = findSchematicComments(app, d.Schematic.ID)
		d.Title = fmt.Sprintf("Comments for %s", d.Schematic.Title)

		html, err := registry.LoadFiles(schematicCommentsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
