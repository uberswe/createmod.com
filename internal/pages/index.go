package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"net/http"
)

const indexTemplate = "index.html"

type IndexData struct {
	DefaultData
	Schematics   []models.Schematic
	Trending     []models.Schematic
	HighestRated []models.Schematic
	Tags         []models.SchematicTagWithCount
}

func IndexHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		schematicsCollection, err := app.Dao().FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.Dao().FindRecordsByFilter(
			schematicsCollection.Id,
			"1=1",
			"-created",
			30,
			0)

		d := IndexData{
			Schematics:   MapResultsToSchematic(app, results),
			Trending:     getTrendingSchematics(app),
			HighestRated: getHighestRatedSchematics(app),
			Tags:         allTagsWithCount(app),
		}
		d.Title = "Create Mod Schematics"
		d.SubCategory = "Home"
		d.Categories = allCategories(app)

		err = c.Render(http.StatusOK, indexTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func getHighestRatedSchematics(app *pocketbase.PocketBase) []models.Schematic {
	var schematics []models.DatabaseSchematic
	err := app.Dao().DB().
		Select("schematics.*", "avg(schematic_ratings.rating) as avg_rating", "count(schematic_ratings.rating) as total_rating").
		From("schematics").
		LeftJoin("schematic_ratings", dbx.NewExp("schematic_ratings.schematic = schematics.id")).
		OrderBy("avg_rating DESC").
		AndOrderBy("total_rating DESC").
		GroupBy("schematics.id").
		Limit(10).
		All(&schematics)
	if err != nil {
		app.Logger().Debug("could not fetch highest rated", "error", err.Error())
		return nil
	}
	return models.DatabaseSchematicsToSchematics(schematics)
}

func getTrendingSchematics(app *pocketbase.PocketBase) []models.Schematic {
	var schematics []models.DatabaseSchematic
	err := app.Dao().DB().
		Select("schematics.*", "avg(schematic_views.count) as avg_views").
		From("schematic_views").
		LeftJoin("schematics", dbx.NewExp("schematic_views.schematic = schematics.id")).
		Where(dbx.NewExp("schematic_views.type = 0")).
		AndWhere(dbx.NewExp("schematic_views.created > (SELECT DATETIME('now', '-1 day'))")).
		OrderBy("avg_views DESC").
		GroupBy("schematics.id").
		Limit(10).
		All(&schematics)
	if err != nil {
		app.Logger().Debug("could not fetch trending", "error", err.Error())
		return nil
	}
	return models.DatabaseSchematicsToSchematics(schematics)
}

func findUserFromID(app *pocketbase.PocketBase, userID string) *models.User {
	userCollection, err := app.Dao().FindCollectionByNameOrId("users")
	if err != nil {
		return nil
	}
	record, err := app.Dao().FindRecordById(userCollection.Id, userID)
	if err != nil || record == nil {
		return nil
	}
	return mapResultToUser(record)
}

func mapResultToUser(record *pbmodels.Record) *models.User {
	return &models.User{
		ID:       record.GetId(),
		Username: record.GetString("username"),
		Avatar:   record.GetString("avatar"),
	}
}
