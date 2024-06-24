package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"html/template"
	"net/http"
	"strings"
)

const schematicTemplate = "schematic.html"

type SchematicData struct {
	DefaultData
	Schematic models.Schematic
}

func SchematicHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.Dao().FindRecordsByFilter(
			schematicsCollection.Id,
			"name = {:name}",
			"-created",
			1,
			0,
			dbx.Params{"name": c.PathParam("name")})

		if len(results) != 1 {
			return c.Render(http.StatusNotFound, fourOhFourTemplate, nil)
		}

		d := SchematicData{
			Schematic: mapResultToSchematic(app, results[0]),
		}
		d.Title = d.Schematic.Title
		d.SubCategory = "Schematic"

		err = c.Render(http.StatusOK, schematicTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func mapResultsToSchematic(app *pocketbase.PocketBase, results []*pbmodels.Record) (schematics []models.Schematic) {
	for i := range results {
		schematics = append(schematics, mapResultToSchematic(app, results[i]))
	}
	return schematics
}

func mapResultToSchematic(app *pocketbase.PocketBase, result *pbmodels.Record) (schematic models.Schematic) {
	return models.Schematic{
		ID:               result.GetId(),
		Created:          result.Created.Time(),
		Author:           findUserFromID(app, result.GetString("author")),
		CommentCount:     result.GetInt("comment_count"),
		CommentStatus:    result.GetBool("comment_status"),
		Content:          result.GetString("content"),
		HTMLContent:      template.HTML(strings.ReplaceAll(template.HTMLEscapeString(result.GetString("content")), "\n", "<br/>")),
		Excerpt:          result.GetString("excerpt"),
		FeaturedImage:    result.GetString("featured_image"),
		Gallery:          result.GetStringSlice("gallery"),
		HasGallery:       len(result.GetStringSlice("gallery")) > 0,
		Title:            result.GetString("title"),
		Name:             result.GetString("name"),
		Video:            result.GetString("video"),
		HasDependencies:  result.GetBool("has_dependencies"),
		Dependencies:     result.GetString("dependencies"),
		HTMLDependencies: template.HTML(strings.ReplaceAll(template.HTMLEscapeString(result.GetString("dependencies")), "\n", "<br/>")),
		Categories:       findCategoriesFromIDs(app, result.GetStringSlice("categories")),
		Tags:             findTagsFromIDs(app, result.GetStringSlice("tags")),
		CreatemodVersion: result.GetString("createmod_version"),
		MinecraftVersion: result.GetString("minecraft_version"),
		Views:            result.GetInt("views"),
	}
}

func findTagsFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicTag {
	tagsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByIds(tagsCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToTags(records)
}

func mapResultToTags(records []*pbmodels.Record) (tags []models.SchematicTag) {
	for i := range records {
		tags = append(tags, models.SchematicTag{
			ID:   records[i].GetId(),
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return tags
}

func findCategoriesFromIDs(app *pocketbase.PocketBase, s []string) []models.SchematicCategory {
	categoriesCollection, err := app.Dao().FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByIds(categoriesCollection.Id, s)
	if err != nil {
		return nil
	}
	return mapResultToCategories(records)
}

func mapResultToCategories(records []*pbmodels.Record) (categories []models.SchematicCategory) {
	for i := range records {
		categories = append(categories, models.SchematicCategory{
			ID:   records[i].GetId(),
			Key:  records[i].GetString("key"),
			Name: records[i].GetString("name"),
		})
	}
	return categories
}
