package pages

import (
	"createmod/internal/models"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
)

const uploadTemplate = "./template/dist/upload.html"

type UploadData struct {
	DefaultData
	MinecraftVersions []models.MinecraftVersion
	CreatemodVersions []models.CreatemodVersion
	Tags              []models.SchematicTag
}

func UploadHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := UploadData{}
		d.Populate(e)
		d.Title = "Upload A Schematic"
		d.Categories = allCategories(app)
		d.Tags = allTags(app)
		d.MinecraftVersions = allMinecraftVersions(app)
		d.CreatemodVersions = allCreatemodVersions(app)
		html, err := registry.LoadFiles(uploadTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func allCreatemodVersions(app *pocketbase.PocketBase) []models.CreatemodVersion {
	createmodVersionCollection, err := app.FindCollectionByNameOrId("createmod_versions")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(createmodVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCreatemodVersions(records)
}

func mapResultToCreatemodVersions(records []*core.Record) []models.CreatemodVersion {
	versions := make([]models.CreatemodVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.CreatemodVersion{
			ID:      r.Id,
			Version: r.GetString("version"),
		})
	}
	return versions
}

func allMinecraftVersions(app *pocketbase.PocketBase) []models.MinecraftVersion {
	minecraftVersionCollection, err := app.FindCollectionByNameOrId("minecraft_versions")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(minecraftVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToMinecraftVersions(records)
}

func mapResultToMinecraftVersions(records []*core.Record) []models.MinecraftVersion {
	versions := make([]models.MinecraftVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.MinecraftVersion{
			ID:      r.Id,
			Version: r.GetString("version"),
		})
	}
	return versions
}
