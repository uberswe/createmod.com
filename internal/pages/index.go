package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"fmt"
	tmpl "html/template"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/drexedam/gravatar"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math"
	"sort"
	"time"
)

var indexTemplates = append([]string{
	"./template/index.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_full.html",
	"./template/include/schematic_card_medium.html",
	"./template/include/schematic_card_small.html",
	"./template/include/schematic_card_featured.html",
}, commonTemplates...)

var indexTabTemplates = append([]string{
	"./template/index_tab.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

const indexPageSize = 12

type IndexData struct {
	DefaultData
	Featured        []models.Schematic
	Schematics      []models.Schematic
	Trending        []models.Schematic
	HighestRated    []models.Schematic
	HasNext         bool // latest tab
	TrendingHasNext bool
	HighestHasNext  bool
}

func IndexHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := e.Request.URL.Query()
		tab := q.Get("tab")
		page := 1
		if p := q.Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		isHTMX := e.Request.Header.Get("HX-Request") != ""

		// HTMX tab request — return just the tab panel partial
		if isHTMX && tab != "" {
			return renderTabPartial(app, cacheService, registry, e, tab, page)
		}

		// Full page load
		schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}

		// Fetch featured schematics
		var featuredSchematics []models.Schematic
		featuredRecords, err := app.FindRecordsByFilter("schematics", "deleted = null && moderated = true && featured = true", "-created", 20, 0)
		if err == nil && len(featuredRecords) > 0 {
			// Shuffle and pick up to 3
			rand.Shuffle(len(featuredRecords), func(i, j int) {
				featuredRecords[i], featuredRecords[j] = featuredRecords[j], featuredRecords[i]
			})
			if len(featuredRecords) > 3 {
				featuredRecords = featuredRecords[:3]
			}
			featuredSchematics = MapResultsToSchematic(app, featuredRecords, cacheService)
		}

		// Latest schematics (first page)
		latestResults, latestHasNext := fetchLatestPage(app, schematicsCollection.Id, 1)

		trendingSchematics, trendingHasNext := getTrendingSchematicsPage(app, cacheService, 1)

		// Pad featured with trending if fewer than 3
		if len(featuredSchematics) < 3 && len(trendingSchematics) > 0 {
			for _, t := range trendingSchematics {
				if len(featuredSchematics) >= 3 {
					break
				}
				found := false
				for _, f := range featuredSchematics {
					if f.ID == t.ID {
						found = true
						break
					}
				}
				if !found {
					featuredSchematics = append(featuredSchematics, t)
				}
			}
		}

		highestRated, highestHasNext := getHighestRatedSchematicsPage(app, cacheService, 1)

		d := IndexData{
			Featured:        featuredSchematics,
			Schematics:      MapResultsToSchematic(app, latestResults, cacheService),
			Trending:        trendingSchematics,
			HighestRated:    highestRated,
			HasNext:         latestHasNext,
			TrendingHasNext: trendingHasNext,
			HighestHasNext:  highestHasNext,
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

// TabData is used for rendering tab partials via HTMX.
type TabData struct {
	DefaultData
	Items   []models.Schematic
	Tab     string
	Page    int
	HasPrev bool
	HasNext bool
	PrevURL string
	NextURL string
}

func renderTabPartial(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, e *core.RequestEvent, tab string, page int) error {
	var items []models.Schematic
	var hasNext bool

	switch tab {
	case "trending":
		items, hasNext = getTrendingSchematicsPage(app, cacheService, page)
	case "highest":
		items, hasNext = getHighestRatedSchematicsPage(app, cacheService, page)
	default:
		tab = "latest"
		col, err := app.FindCollectionByNameOrId("schematics")
		if err != nil {
			return err
		}
		var results []*core.Record
		results, hasNext = fetchLatestPage(app, col.Id, page)
		items = MapResultsToSchematic(app, results, cacheService)
	}

	d := TabData{
		Items:   items,
		Tab:     tab,
		Page:    page,
		HasPrev: page > 1,
		HasNext: hasNext,
	}
	if d.HasPrev {
		d.PrevURL = fmt.Sprintf("/?tab=%s&p=%d", tab, page-1)
	}
	if d.HasNext {
		d.NextURL = fmt.Sprintf("/?tab=%s&p=%d", tab, page+1)
	}
	d.Populate(e)

	html, err := registry.LoadFiles(indexTabTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

func fetchLatestPage(app *pocketbase.PocketBase, collectionID string, page int) ([]*core.Record, bool) {
	limit := indexPageSize + 1
	offset := (page - 1) * indexPageSize
	results, err := app.FindRecordsByFilter(
		collectionID,
		"deleted = null && moderated = true && (scheduled_at = null || scheduled_at <= {:now})",
		"-created",
		limit,
		offset,
		dbx.Params{"now": time.Now()},
	)
	if err != nil {
		return nil, false
	}
	hasNext := len(results) > indexPageSize
	if hasNext {
		results = results[:indexPageSize]
	}
	return results, hasNext
}

func getHighestRatedSchematicsPage(app *pocketbase.PocketBase, cacheService *cache.Service, page int) ([]models.Schematic, bool) {
	limit := indexPageSize + 1
	offset := (page - 1) * indexPageSize

	var res []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*", "avg(schematic_ratings.rating) as avg_rating", "count(schematic_ratings.rating) as total_rating").
		From("schematics").
		LeftJoin("schematic_ratings", dbx.NewExp("schematic_ratings.schematic = schematics.id")).
		Where(dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL) AND schematics.moderated = 1 AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
		OrderBy("avg_rating DESC").
		AndOrderBy("total_rating DESC").
		GroupBy("schematics.id").
		Having(dbx.NewExp("count(schematic_ratings.rating) > 0")).
		Limit(int64(limit)).
		Offset(int64(offset)).
		All(&res)
	if err != nil {
		app.Logger().Debug("could not fetch highest rated", "error", err.Error())
		return nil, false
	}
	hasNext := len(res) > indexPageSize
	if hasNext {
		res = res[:indexPageSize]
	}
	return MapResultsToSchematic(app, res, cacheService), hasNext
}

func getHighestRatedSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	results, _ := getHighestRatedSchematicsPage(app, cacheService, 1)
	return results
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

// ComputeTrendingScores computes trending scores for all recent schematics.
// Returns a map of schematic ID to score. This can be passed to search.Service.SetTrendingScores().
func ComputeTrendingScores(app *pocketbase.PocketBase) map[string]float64 {
	now := time.Now().UTC()
	var schemRecs []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL) AND schematics.moderated = 1 AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
		OrderBy("created DESC").
		Limit(200).
		All(&schemRecs)
	if err != nil || len(schemRecs) == 0 {
		return nil
	}

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

	const decay = 1.8
	const ratingWeight = 2.0
	scores := make(map[string]float64, len(schemRecs))
	for _, rec := range schemRecs {
		id := rec.Id
		created := rec.GetDateTime("created").Time()
		v := views48[id]
		r := ratingsSum[id]
		scores[id] = trendingScore(now, created, v, r, decay, ratingWeight)
	}
	return scores
}

// getAllTrendingSchematics returns the full sorted trending list (up to 200 candidates).
func getAllTrendingSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	cached, found := cacheService.GetSchematics(cache.TrendingSchematicsKey)
	if found {
		return cached
	}

	now := time.Now().UTC()
	var schemRecs []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL) AND schematics.moderated = 1 AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
		OrderBy("created DESC").
		Limit(200).
		All(&schemRecs)
	if err != nil || len(schemRecs) == 0 {
		return nil
	}

	// Aggregate recent views (last 48h)
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

	// Aggregate ratings sum
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

	// Compute scores, sort
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

	ordered := make([]*core.Record, 0, len(scoredList))
	for _, it := range scoredList {
		ordered = append(ordered, it.rec)
	}

	all := MapResultsToSchematic(app, ordered, cacheService)
	cacheService.SetSchematics(cache.TrendingSchematicsKey, all)
	return all
}

func getTrendingSchematicsPage(app *pocketbase.PocketBase, cacheService *cache.Service, page int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematics(app, cacheService)
	offset := (page - 1) * indexPageSize
	if offset >= len(all) {
		return nil, false
	}
	end := offset + indexPageSize
	hasNext := end < len(all)
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], hasNext
}

// getTrendingSchematics returns the first page of trending for backward compatibility.
func getTrendingSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	results, _ := getTrendingSchematicsPage(app, cacheService, 1)
	return results
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
