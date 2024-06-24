package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"net/http"
)

const indexTemplate = "index.html"

type IndexData struct {
	DefaultData
	Schematics []models.Schematic
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
			Schematics: mapResultsToSchematic(app, results),
		}
		d.Title = "Create Mod Schematics"
		d.SubCategory = "Home"

		err = c.Render(http.StatusOK, indexTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
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
