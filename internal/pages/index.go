package pages

import (
	"createmod/internal/models"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	pbmodels "github.com/pocketbase/pocketbase/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
	"time"
)

const indexTemplate = "index.html"

// TODO consider implementing a cache service which handles in-memory storage
var (
	trendingCacheTime      time.Time
	trendingSchematics     []models.Schematic
	highestRatedCacheTime  time.Time
	highestRatedSchematics []models.Schematic
)

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
			50,
			0)

		d := IndexData{
			Schematics:   MapResultsToSchematic(app, results),
			Trending:     getTrendingSchematics(app),
			HighestRated: getHighestRatedSchematics(app),
			Tags:         allTagsWithCount(app),
		}
		d.Populate(c)
		d.Title = "Minecraft Schematics"
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
	if len(highestRatedSchematics) > 0 && time.Now().Before(highestRatedCacheTime.Add(time.Hour*24)) {
		return highestRatedSchematics
	}
	// TODO a field for average rating can be aggregated daily and indexed to improve performance
	// Also consider if this is a good metric, perhaps adding more weight to the number of ratings could be good.
	// Currently it takes an the average, perhaps we should use the mean rating instead as this would account for number of ratings?
	var res []*pbmodels.Record
	err := app.Dao().RecordQuery("schematics").
		Select("schematics.*", "avg(schematic_ratings.rating) as avg_rating", "count(schematic_ratings.rating) as total_rating").
		From("schematics").
		LeftJoin("schematic_ratings", dbx.NewExp("schematic_ratings.schematic = schematics.id")).
		OrderBy("avg_rating DESC").
		AndOrderBy("total_rating DESC").
		GroupBy("schematics.id").
		Limit(10).
		All(&res)
	if err != nil {
		app.Logger().Debug("could not fetch highest rated", "error", err.Error())
		return nil
	}
	highestRatedSchematics = MapResultsToSchematic(app, res)
	highestRatedCacheTime = time.Now()
	return highestRatedSchematics
}

func getTrendingSchematics(app *pocketbase.PocketBase) []models.Schematic {
	if len(trendingSchematics) > 0 && time.Now().Before(trendingCacheTime.Add(time.Hour*24)) {
		return trendingSchematics
	}
	// TODO a field for daily and weekly views can be aggregated daily and indexed to improve performance
	var res []*pbmodels.Record
	err := app.Dao().RecordQuery("schematics").
		Select("schematics.*", "avg(schematic_views.count) as avg_views").
		From("schematic_views").
		LeftJoin("schematics", dbx.NewExp("schematic_views.schematic = schematics.id")).
		Where(dbx.NewExp("schematic_views.type = 0")).
		AndWhere(dbx.NewExp("schematic_views.created > (SELECT DATETIME('now', '-1 day'))")).
		OrderBy("avg_views DESC").
		GroupBy("schematics.id").
		Limit(10).
		All(&res)
	if err != nil {
		app.Logger().Debug("could not fetch trending", "error", err.Error())
		return nil
	}
	trendingSchematics = MapResultsToSchematic(app, res)
	trendingCacheTime = time.Now()
	return trendingSchematics
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

	caser := cases.Title(language.English)
	return &models.User{
		ID:       record.GetId(),
		Username: caser.String(record.GetString("username")),
		Avatar:   record.GetString("avatar"),
	}
}
