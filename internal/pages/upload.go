package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"net/http"
)

const uploadTemplate = "upload.html"

type UploadData struct {
	DefaultData
	MinecraftVersions []models.MinecraftVersion
	CreatemodVersions []models.CreatemodVersion
	Tags              []models.SchematicTag
}

func UploadHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		d := UploadData{}
		d.Populate(c)
		d.Title = "Upload A Schematic"
		d.Categories = allCategories(app)
		d.Tags = allTags(app)
		d.MinecraftVersions = allMinecraftVersions(app)
		d.CreatemodVersions = allCreatemodVersions(app)
		err := c.Render(http.StatusOK, uploadTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func allCreatemodVersions(app *pocketbase.PocketBase) []models.CreatemodVersion {
	createmodVersionCollection, err := app.Dao().FindCollectionByNameOrId("createmod_versions")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByFilter(createmodVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCreatemodVersions(records)
}

func mapResultToCreatemodVersions(records []*pbmodels.Record) []models.CreatemodVersion {
	versions := make([]models.CreatemodVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.CreatemodVersion{
			ID:      r.GetId(),
			Version: r.GetString("version"),
		})
	}
	return versions
}

func allMinecraftVersions(app *pocketbase.PocketBase) []models.MinecraftVersion {
	minecraftVersionCollection, err := app.Dao().FindCollectionByNameOrId("minecraft_versions")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByFilter(minecraftVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToMinecraftVersions(records)
}

func mapResultToMinecraftVersions(records []*pbmodels.Record) []models.MinecraftVersion {
	versions := make([]models.MinecraftVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.MinecraftVersion{
			ID:      r.GetId(),
			Version: r.GetString("version"),
		})
	}
	return versions
}
