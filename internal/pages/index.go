package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/drexedam/gravatar"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	tmpl "html/template"
	"net/http"
)

var indexTemplates = []string{
	"./template/dist/index.html",
	"./template/dist/include/schematic_card.html",
}

type IndexData struct {
	DefaultData
	Schematics   []models.Schematic
	Trending     []models.Schematic
	HighestRated []models.Schematic
	Tags         []models.SchematicTagWithCount
}

func IndexHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		results, err := app.FindRecordsByFilter(
			schematicsCollection.Id,
			"deleted = null",
			"-created",
			50,
			0)

		d := IndexData{
			Schematics:   MapResultsToSchematic(app, results, cacheService),
			Trending:     getTrendingSchematics(app, cacheService),
			HighestRated: getHighestRatedSchematics(app, cacheService),
			Tags:         allTagsWithCount(app, cacheService),
		}
		d.Populate(e)
		d.Title = "Minecraft Schematics"
		d.Description = "The Create Schematics Repository. Download user-created Create Mod Schematics. Upload your own for others to see."
		d.Slug = "/"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.SubCategory = "Home"
		d.Categories = allCategories(app, cacheService)

		html, err := registry.LoadFiles(indexTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func getHighestRatedSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	highestRatedSchematics, found := cacheService.GetSchematics(cache.HighestRatedSchematicsKey)
	if found {
		return highestRatedSchematics
	}
	// TODO a field for average rating can be aggregated daily and indexed to improve performance
	// Also consider if this is a good metric, perhaps adding more weight to the number of ratings could be good.
	// Currently it takes an the average, perhaps we should use the mean rating instead as this would account for number of ratings?
	var res []*core.Record
	err := app.RecordQuery("schematics").
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
	highestRatedSchematics = MapResultsToSchematic(app, res, cacheService)
	cacheService.SetSchematics(cache.HighestRatedSchematicsKey, highestRatedSchematics)
	return highestRatedSchematics
}

func getTrendingSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	trendingSchematics, found := cacheService.GetSchematics(cache.TrendingSchematicsKey)
	if found {
		return trendingSchematics
	}
	// TODO a field for daily and weekly views can be aggregated daily and indexed to improve performance
	var res []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*", "avg(schematic_views.count) as avg_views").
		From("schematic_views").
		LeftJoin("schematics", dbx.NewExp("schematic_views.schematic = schematics.id")).
		Where(dbx.NewExp("schematic_views.type = 0")).
		AndWhere(dbx.NewExp("schematic_views.created > (SELECT DATETIME('now', '-2 day'))")).
		OrderBy("avg_views DESC").
		GroupBy("schematics.id").
		Limit(10).
		All(&res)
	if err != nil {
		app.Logger().Debug("could not fetch trending", "error", err.Error())
		return nil
	}
	trendingSchematics = MapResultsToSchematic(app, res, cacheService)
	cacheService.SetSchematics(cache.TrendingSchematicsKey, trendingSchematics)
	return trendingSchematics
}

func findUserFromID(app *pocketbase.PocketBase, userID string) *models.User {
	userCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil
	}
	record, err := app.FindRecordById(userCollection.Id, userID)
	if err != nil || record == nil {
		return nil
	}
	return mapResultToUser(record)
}

func mapResultToUser(record *core.Record) *models.User {
	caser := cases.Title(language.English)
	avatarUrl := gravatar.New(record.GetString("email")).
		Size(200).
		Default(gravatar.MysteryMan).
		Rating(gravatar.Pg).
		AvatarURL()
	return &models.User{
		ID:        record.Id,
		Username:  caser.String(record.GetString("username")),
		Avatar:    tmpl.URL(avatarUrl),
		HasAvatar: len(avatarUrl) > 0,
	}
}
