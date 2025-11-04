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
	"math"
	"net/http"
	"sort"
	"time"
)

var indexTemplates = append([]string{
	"./template/index.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_full.html",
	"./template/include/schematic_card_medium.html",
}, commonTemplates...)

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
			"deleted = null && moderated = true && (scheduled_at = null || scheduled_at <= {:now})",
			"-created",
			50,
			0,
			dbx.Params{"now": time.Now()},
		)

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
		Where(dbx.NewExp("schematics.deleted = null AND schematics.moderated = true AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
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

// trendingScore computes a decayed popularity score combining recent views and ratings sum.
// score = (views48h + ratingWeight*log1p(ratingsSum)) / pow(ageHours+2, decay)
func trendingScore(now time.Time, created time.Time, views48h float64, ratingsSum float64, decay float64, ratingWeight float64) float64 {
	ageHours := now.Sub(created).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	den := math.Pow(ageHours+2, decay)
	if den == 0 {
		den = 1
	}
	return (views48h + ratingWeight*math.Log1p(ratingsSum)) / den
}

func getTrendingSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	trendingSchematics, found := cacheService.GetSchematics(cache.TrendingSchematicsKey)
	if found {
		return trendingSchematics
	}

	now := time.Now().UTC()
	// 1) Fetch eligible schematics (moderated, not deleted, scheduled ok)
	var schemRecs []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("schematics.deleted = null AND schematics.moderated = true AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
		OrderBy("-created").
		Limit(200).
		All(&schemRecs)
	if err != nil || len(schemRecs) == 0 {
		app.Logger().Debug("trending: fetch schematics failed or none, falling back", "err", func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}())
		return nil
	}

	// 2) Aggregate recent views (last 48h)
	twoDaysAgo := now.Add(-48 * time.Hour)
	twoDaysAgoStr := twoDaysAgo.Format("2006-01-02 15:04:05")
	type kv struct {
		ID string
		V  float64
	}
	var viewRows []kv
	err = app.RecordQuery("schematic_views").
		Select("schematic as id", "SUM(count) as v").
		From("schematic_views").
		Where(dbx.NewExp("type = 0 AND created > {:ts}", dbx.Params{"ts": twoDaysAgoStr})).
		GroupBy("schematic").
		All(&viewRows)
	views48 := make(map[string]float64, len(viewRows))
	if err == nil {
		for _, r := range viewRows {
			views48[r.ID] = r.V
		}
	}

	// 3) Aggregate ratings sum
	var ratingRows []kv
	err2 := app.RecordQuery("schematic_ratings").
		Select("schematic as id", "SUM(rating) as v").
		From("schematic_ratings").
		GroupBy("schematic").
		All(&ratingRows)
	ratingsSum := make(map[string]float64, len(ratingRows))
	if err2 == nil {
		for _, r := range ratingRows {
			ratingsSum[r.ID] = r.V
		}
	}

	// 4) Compute scores, sort
	type scored struct {
		rec   *core.Record
		score float64
	}
	scoredList := make([]scored, 0, len(schemRecs))
	const decay = 1.8
	const ratingWeight = 2.0
	for _, rec := range schemRecs {
		id := rec.Id
		created := rec.GetDateTime("created").Time()
		v := views48[id]
		r := ratingsSum[id]
		s := trendingScore(now, created, v, r, decay, ratingWeight)
		scoredList = append(scoredList, scored{rec: rec, score: s})
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].score > scoredList[j].score })
	if len(scoredList) > 10 {
		scoredList = scoredList[:10]
	}
	ordered := make([]*core.Record, 0, len(scoredList))
	for _, it := range scoredList {
		ordered = append(ordered, it.rec)
	}

	trendingSchematics = MapResultsToSchematic(app, ordered, cacheService)
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
