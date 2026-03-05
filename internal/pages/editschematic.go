package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/store"
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"createmod/internal/server"
	"net/http"
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
}

type SchematicTagWithSelected struct {
	models.SchematicTag
	Selected bool
}

func EditSchematicHandler(searchService *search.Service, cacheService *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		storeSchematic, err := appStore.Schematics.GetByName(context.Background(), name)
		if err != nil || storeSchematic == nil {
			nd := DefaultData{}
			nd.Populate(e)
			nd.Title = i18n.T(nd.Language, "Page Not Found")
			html, err := registry.LoadFiles(fourOhFourTemplates...).Render(nd)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d := EditSchematicData{
			Schematic: MapStoreSchematicToModel(appStore, *storeSchematic, cacheService),
		}
		d.Populate(e)
		d.Title = fmt.Sprintf("%s %s", i18n.T(d.Language, "Editing"), d.Schematic.Title)
		d.Slug = fmt.Sprintf("/schematics/%s/edit", d.Schematic.Name)
		d.Description = strip.StripTags(d.Schematic.Content)
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematic.ID, d.Schematic.FeaturedImage)
		d.SubCategory = "Schematic"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Tags = allTagsFromStore(appStore)
		d.MinecraftVersions = allMinecraftVersionsFromStore(appStore)
		d.CreatemodVersions = allCreatemodVersionsFromStore(appStore)
		d.IsAuthor = d.Schematic.Author.ID == d.UserID
		if storeSchematic.CreatemodVersionID != nil {
			d.CreateModVersionId = *storeSchematic.CreatemodVersionID
		}

		for _, t := range d.Tags {
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
		html, err := registry.LoadFiles(editSchematicTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
