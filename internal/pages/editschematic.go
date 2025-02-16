package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/search"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	template2 "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

var editSchematicTemplates = []string{
	"./template/dist/editschematic.html",
}

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

func EditSchematicHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template2.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name}",
			"-created",
			1,
			0,
			dbx.Params{"name": e.Request.PathValue("name")})

		if len(results) != 1 {
			html, err := registry.LoadFiles(fourOhFourTemplate).Render(nil)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d := EditSchematicData{
			Schematic: mapResultToSchematic(app, results[0], cacheService),
		}
		d.Populate(e)
		d.Title = fmt.Sprintf("Editing %s", d.Schematic.Title)
		d.Slug = fmt.Sprintf("schematics/%s/edit", d.Schematic.Name)
		d.Description = d.Schematic.Content
		d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematic.ID, d.Schematic.FeaturedImage)
		d.SubCategory = "Schematic"
		d.Categories = allCategories(app, cacheService)
		d.Tags = allTags(app)
		d.MinecraftVersions = allMinecraftVersions(app)
		d.CreatemodVersions = allCreatemodVersions(app)
		d.IsAuthor = d.Schematic.Author.ID == d.UserID
		d.CreateModVersionId = results[0].GetString("createmod_version")

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

		countSchematicView(app, results[0])
		html, err := registry.LoadFiles(editSchematicTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
