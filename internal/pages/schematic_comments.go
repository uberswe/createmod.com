package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/i18n"
	"createmod/internal/search"
	"createmod/internal/store"
	"fmt"
	"createmod/internal/server"
	"net/http"
)

var schematicCommentsTemplates = []string{
	"./template/include/comments.html",
}

// SchematicCommentsHandler returns only the comments list for a schematic.
// Useful for HTMX partial refresh of comments.
func SchematicCommentsHandler(searchService *search.Service, cacheService *cache.Service, registry *server.Registry, _ *discord.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		storeSchematic, err := appStore.Schematics.GetByName(context.Background(), name)
		if err != nil || storeSchematic == nil || (storeSchematic.Deleted != nil && !storeSchematic.Deleted.IsZero()) {
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
			Schematic: MapStoreSchematicToModel(appStore, *storeSchematic, cacheService),
		}
		d.Populate(e)
		d.Comments = findSchematicCommentsFromStore(appStore, d.Schematic.ID)
		d.Title = fmt.Sprintf("Comments for %s", d.Schematic.Title)

		html, err := registry.LoadFiles(schematicCommentsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
