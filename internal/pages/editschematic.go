package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"net/http"
	"net/url"
)

var editSchematicTemplates = append([]string{
	"./template/editschematic.html",
}, commonTemplates...)

type EditSchematicData struct {
	DefaultData
	Schematic     models.Schematic
	AuthorHasMore bool
	// IsAuthor of the current schematic, for edit and delete actions
	IsAuthor           bool
	MinecraftVersions  []models.MinecraftVersion
	CreatemodVersions  []models.CreatemodVersion
	Tags               []models.SchematicTag
	TagsWithSelected   []SchematicTagWithSelected
	CreateModVersionId string
	MinecraftVersionId string
}

type SchematicTagWithSelected struct {
	models.SchematicTag
	Selected bool
}

func EditSchematicHandler(searchEngine search.SearchEngine, cacheService *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		storeSchematic, err := appStore.Schematics.GetByName(context.Background(), name)
		if err != nil || storeSchematic == nil {
			return RenderNotFound(registry, searchEngine, cacheService, appStore, e)
		}

		d := EditSchematicData{
			Schematic: MapStoreSchematicToModel(appStore, *storeSchematic, cacheService),
		}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Schematics"), "/schematics", d.Schematic.Title, "/schematics/"+d.Schematic.Name, i18n.T(d.Language, "Edit"))
		d.Title = fmt.Sprintf("%s %s", i18n.T(d.Language, "Editing"), d.Schematic.Title)
		d.Slug = fmt.Sprintf("/schematics/%s/edit", d.Schematic.Name)
		d.Description = strip.StripTags(d.Schematic.Content)
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematic.ID, url.PathEscape(d.Schematic.FeaturedImage))
		d.SubCategory = "Schematic"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Tags = allTagsFromStore(appStore)
		d.MinecraftVersions = allMinecraftVersionsFromStore(appStore)
		d.CreatemodVersions = allCreatemodVersionsFromStore(appStore)
		d.IsAuthor = d.Schematic.Author.ID == d.UserID
		if storeSchematic.CreatemodVersionID != nil {
			d.CreateModVersionId = *storeSchematic.CreatemodVersionID
		}
		if storeSchematic.MinecraftVersionID != nil {
			d.MinecraftVersionId = *storeSchematic.MinecraftVersionID
		}

		// Build tag list with selected state. Start from all public tags.
		publicKeySet := make(map[string]bool, len(d.Tags))
		for _, t := range d.Tags {
			publicKeySet[t.Key] = true
			selected := false
			for _, t2 := range d.Schematic.Tags {
				if t.Key == t2.Key {
					selected = true
				}
			}
			d.TagsWithSelected = append(d.TagsWithSelected, SchematicTagWithSelected{
				SchematicTag: t,
				Selected:     selected,
			})
		}
		// Include any schematic tags not already in the public list (e.g. pending tags).
		for _, t := range d.Schematic.Tags {
			if !publicKeySet[t.Key] {
				d.TagsWithSelected = append(d.TagsWithSelected, SchematicTagWithSelected{
					SchematicTag: t,
					Selected:     true,
				})
			}
		}
		html, err := registry.LoadFiles(editSchematicTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
