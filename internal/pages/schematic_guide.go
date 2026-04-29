package pages

import (
	stdctx "context"
	"io"
	"net/http"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/nbtparser"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

var schematicGuideTemplates = append([]string{
	"./template/schematic-guide.html",
}, commonTemplates...)

type SchematicGuideData struct {
	DefaultData
	Schematic     store.Schematic
	SchematicSlug string
}

func SchematicGuideHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := stdctx.Background()
		name := e.Request.PathValue("name")

		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil || s.ModerationState == store.ModerationDeleted {
			return e.NotFoundError("schematic not found", nil)
		}

		d := SchematicGuideData{}
		d.Populate(e)
		d.Schematic = *s
		d.SchematicSlug = name
		d.Title = i18n.T(d.Language, "Building Guide") + " — " + s.Title
		d.Description = "Layer by layer building guide for " + s.Title
		d.Slug = "/schematics/" + name + "/guide"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, s.Title, "/schematics/"+name, "Guide")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(schematicGuideTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func SchematicGuideAPIHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := stdctx.Background()
		name := e.Request.PathValue("name")

		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil || s.ModerationState == store.ModerationDeleted {
			return e.NotFoundError("schematic not found", nil)
		}

		if s.SchematicFile == "" {
			return e.BadRequestError("schematic has no file", nil)
		}

		if storageSvc == nil {
			return e.InternalServerError("storage not configured", nil)
		}

		reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, s.SchematicFile)
		if err != nil {
			return e.InternalServerError("failed to download schematic file", nil)
		}
		defer reader.Close()

		data, err := io.ReadAll(io.LimitReader(reader, 10*1024*1024))
		if err != nil {
			return e.InternalServerError("failed to read schematic file", nil)
		}

		guide, err := nbtparser.ExtractGuideBlocks(data)
		if err != nil {
			return e.BadRequestError("failed to parse schematic: "+err.Error(), nil)
		}

		return e.JSON(http.StatusOK, guide)
	}
}
