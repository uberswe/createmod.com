package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"fmt"
	tmpl "html/template"
	"net/http"
	"strconv"
	"strings"

	"math"
	"sort"
	"time"

	"github.com/drexedam/gravatar"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var indexTemplates = append([]string{
	"./template/index.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_full.html",
	"./template/include/schematic_card_medium.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

var indexTabTemplates = append([]string{
	"./template/index_tab.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

const indexPageSize = 8

// CategorySection holds one category's schematics for the index page.
type CategorySection struct {
	Category models.SchematicCategory
	Items    []models.Schematic
	HasNext  bool
}

type IndexData struct {
	DefaultData
	Schematics       []models.Schematic
	Trending         []models.Schematic
	HighestRated     []models.Schematic
	HasNext          bool // latest tab
	TrendingHasNext  bool
	HighestHasNext   bool
	CategorySections []CategorySection
}

func IndexHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
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
			return renderTabPartial(app, cacheService, registry, appStore, e, tab, page)
		}

		// Full page load — serve from pre-warmed cache when available.
		latestSchematics, latestCached := cacheService.GetSchematics(cache.LatestSchematicsKey)
		trendingSchematics, _ := cacheService.GetSchematics(cache.TrendingSchematicsKey)
		highestRated, _ := cacheService.GetSchematics(cache.HighestRatedSchematicsKey)

		// The trending cache stores the full sorted list; slice to page 1.
		if len(trendingSchematics) > indexPageSize {
			trendingSchematics = trendingSchematics[:indexPageSize]
		}

		// Determine hasNext flags from cache (default false if not cached)
		latestHasNext := false
		trendingHasNext := false
		highestHasNext := false
		if v, ok := cacheService.Get(cache.LatestHasNextKey); ok {
			if b, ok := v.(bool); ok {
				latestHasNext = b
			}
		}
		if v, ok := cacheService.Get(cache.TrendingHasNextKey); ok {
			if b, ok := v.(bool); ok {
				trendingHasNext = b
			}
		}
		if v, ok := cacheService.Get(cache.HighestRatedHasNextKey); ok {
			if b, ok := v.(bool); ok {
				highestHasNext = b
			}
		}

		// Fallback: if cache is cold (first request before warm completes), query directly
		if !latestCached {
			schematicsCollection, err := app.FindCollectionByNameOrId("schematics")
			if err != nil {
				return err
			}
			var latestResults []*core.Record
			latestResults, latestHasNext = fetchLatestPage(app, schematicsCollection.Id, 1)
			latestSchematics = MapResultsToSchematic(app, latestResults, cacheService)
			trendingSchematics, trendingHasNext = getTrendingSchematicsPage(app, cacheService, 1)
			highestRated, highestHasNext = getHighestRatedSchematicsPage(app, cacheService, 1)
		}

		// Build category sections
		categories := allCategoriesFromStore(appStore, app, cacheService)
		categorySections := make([]CategorySection, 0, len(categories))
		for _, cat := range categories {
			cacheKey := fmt.Sprintf("CategorySection:%s", cat.Key)
			cacheHasNextKey := fmt.Sprintf("CategorySectionHasNext:%s", cat.Key)
			items, cached := cacheService.GetSchematics(cacheKey)
			catHasNext := false
			if cached {
				if v, ok := cacheService.Get(cacheHasNextKey); ok {
					if b, ok := v.(bool); ok {
						catHasNext = b
					}
				}
			} else {
				items, catHasNext = getCategoryTrendingPage(app, cacheService, cat.ID, 1)
			}
			categorySections = append(categorySections, CategorySection{
				Category: cat,
				Items:    items,
				HasNext:  catHasNext,
			})
		}

		d := IndexData{
			Schematics:       latestSchematics,
			Trending:         trendingSchematics,
			HighestRated:     highestRated,
			HasNext:          latestHasNext,
			TrendingHasNext:  trendingHasNext,
			HighestHasNext:   highestHasNext,
			CategorySections: categorySections,
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "page.index.title")
		d.Description = i18n.T(d.Language, "page.index.description")
		d.Slug = "/"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.SubCategory = "Home"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

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

func renderTabPartial(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, appStore *store.Store, e *core.RequestEvent, tab string, page int) error {
	var items []models.Schematic
	var hasNext bool

	switch {
	case tab == "trending":
		items, hasNext = getTrendingSchematicsPage(app, cacheService, page)
	case tab == "highest":
		items, hasNext = getHighestRatedSchematicsPage(app, cacheService, page)
	case strings.HasPrefix(tab, "cat-"):
		catKey := strings.TrimPrefix(tab, "cat-")
		categories := allCategoriesFromStore(appStore, app, cacheService)
		var catID string
		for _, c := range categories {
			if c.Key == catKey {
				catID = c.ID
				break
			}
		}
		if catID == "" {
			return e.NotFoundError("", nil)
		}
		items, hasNext = getCategoryTrendingPage(app, cacheService, catID, page)
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
		"deleted = '' && moderated = true && (scheduled_at = null || scheduled_at <= {:now})",
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

// trendingEpoch is a fixed reference point for the Reddit-style hot score.
// All scores are relative to this; the exact value doesn't matter as long as it's consistent.
var trendingEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// trendingTimescale controls how quickly newer content displaces older content.
// Every timescale period, an item needs 10x more engagement to hold its ranking.
// 365 days in seconds — gives older high-quality content longer shelf life.
const trendingTimescale = 365 * 24 * 3600.0

// trendingScore computes a Reddit-style hot score.
//
// score = log10(max(engagement, 1)) + (created - epoch) / timescale
//
// Engagement is an additive combination of recent views, total views (dampened),
// rating count (deliberate user action), and rating sum (quality signal).
// When engagement is zero, log10(1) = 0, so items sort purely by creation time (newest first).
// As engagement grows logarithmically, popular items get boosted above their age peers.
func trendingScore(created time.Time, recentViews float64, totalViews float64, ratingCount float64, ratingSum float64, recentDownloads float64, totalDownloads float64) float64 {
	engagement := recentViews + 0.5*math.Log1p(totalViews) + 3.0*ratingCount + ratingSum + 2.0*recentDownloads + 0.5*math.Log1p(totalDownloads)
	order := math.Log10(math.Max(engagement, 1))
	seconds := created.Sub(trendingEpoch).Seconds()
	return order + seconds/trendingTimescale
}

// ComputeTrendingScores computes trending scores for all moderated schematics.
// Returns a map of schematic ID to score. This can be passed to search.Service.SetTrendingScores().
func ComputeTrendingScores(app *pocketbase.PocketBase) map[string]float64 {
	engagement := fetchEngagementData(app)
	if engagement == nil {
		return nil
	}
	return engagement.computeScores()
}

// engagementData holds the pre-fetched data needed to compute trending scores.
type engagementData struct {
	schemRecs       []*core.Record
	recentViews     map[string]float64 // 48h views
	totalViews      map[string]float64 // all-time views
	ratingSum       map[string]float64
	ratingCount     map[string]float64
	recentDownloads map[string]float64 // 48h downloads
	totalDownloads  map[string]float64 // all-time downloads
}

func (ed *engagementData) computeScores() map[string]float64 {
	scores := make(map[string]float64, len(ed.schemRecs))
	for _, rec := range ed.schemRecs {
		id := rec.Id
		created := rec.GetDateTime("created").Time()
		scores[id] = trendingScore(created, ed.recentViews[id], ed.totalViews[id], ed.ratingCount[id], ed.ratingSum[id], ed.recentDownloads[id], ed.totalDownloads[id])
	}
	return scores
}

// fetchEngagementData queries all the signals needed for trending in bulk.
func fetchEngagementData(app *pocketbase.PocketBase) *engagementData {
	var schemRecs []*core.Record
	err := app.RecordQuery("schematics").
		Select("schematics.*").
		From("schematics").
		Where(dbx.NewExp("(schematics.deleted = '' OR schematics.deleted IS NULL) AND schematics.moderated = 1 AND (schematics.scheduled_at IS NULL OR schematics.scheduled_at <= DATETIME('now'))")).
		OrderBy("created DESC").
		All(&schemRecs)
	if err != nil || len(schemRecs) == 0 {
		return nil
	}

	type kv struct {
		ID string
		V  float64
	}

	// Recent views (last 30 days — matches the 12-month trending timescale)
	recentCutoff := time.Now().UTC().Add(-30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	var viewRows []kv
	err = app.RecordQuery("schematic_views").
		Select("schematic as id", "SUM(count) as v").
		From("schematic_views").
		Where(dbx.NewExp("type = 0 AND created > {:ts}", dbx.Params{"ts": recentCutoff})).
		GroupBy("schematic").
		All(&viewRows)
	recentViews := make(map[string]float64, len(viewRows))
	if err == nil {
		for _, r := range viewRows {
			recentViews[r.ID] = r.V
		}
	}

	// All-time total views
	var totalViewRows []kv
	err = app.RecordQuery("schematic_views").
		Select("schematic as id", "SUM(count) as v").
		From("schematic_views").
		Where(dbx.NewExp("type = 0")).
		GroupBy("schematic").
		All(&totalViewRows)
	totalViews := make(map[string]float64, len(totalViewRows))
	if err == nil {
		for _, r := range totalViewRows {
			totalViews[r.ID] = r.V
		}
	}

	// Rating sum
	var ratingSumRows []kv
	err = app.RecordQuery("schematic_ratings").
		Select("schematic as id", "SUM(rating) as v").
		From("schematic_ratings").
		GroupBy("schematic").
		All(&ratingSumRows)
	ratingSum := make(map[string]float64, len(ratingSumRows))
	if err == nil {
		for _, r := range ratingSumRows {
			ratingSum[r.ID] = r.V
		}
	}

	// Rating count
	var ratingCountRows []kv
	err = app.RecordQuery("schematic_ratings").
		Select("schematic as id", "COUNT(rating) as v").
		From("schematic_ratings").
		GroupBy("schematic").
		All(&ratingCountRows)
	ratingCount := make(map[string]float64, len(ratingCountRows))
	if err == nil {
		for _, r := range ratingCountRows {
			ratingCount[r.ID] = r.V
		}
	}

	// Recent downloads (last 48h)
	var recentDlRows []kv
	err = app.RecordQuery("schematic_downloads").
		Select("schematic as id", "SUM(count) as v").
		From("schematic_downloads").
		Where(dbx.NewExp("type = 0 AND created > {:ts}", dbx.Params{"ts": recentCutoff})).
		GroupBy("schematic").
		All(&recentDlRows)
	recentDownloads := make(map[string]float64, len(recentDlRows))
	if err == nil {
		for _, r := range recentDlRows {
			recentDownloads[r.ID] = r.V
		}
	}

	// All-time total downloads
	var totalDlRows []kv
	err = app.RecordQuery("schematic_downloads").
		Select("schematic as id", "SUM(count) as v").
		From("schematic_downloads").
		Where(dbx.NewExp("type = 0")).
		GroupBy("schematic").
		All(&totalDlRows)
	totalDownloads := make(map[string]float64, len(totalDlRows))
	if err == nil {
		for _, r := range totalDlRows {
			totalDownloads[r.ID] = r.V
		}
	}

	return &engagementData{
		schemRecs:       schemRecs,
		recentViews:     recentViews,
		totalViews:      totalViews,
		ratingSum:       ratingSum,
		ratingCount:     ratingCount,
		recentDownloads: recentDownloads,
		totalDownloads:  totalDownloads,
	}
}

// getAllTrendingSchematics returns the full sorted trending list for all moderated schematics.
func getAllTrendingSchematics(app *pocketbase.PocketBase, cacheService *cache.Service) []models.Schematic {
	cached, found := cacheService.GetSchematics(cache.TrendingSchematicsKey)
	if found {
		return cached
	}

	ed := fetchEngagementData(app)
	if ed == nil {
		return nil
	}

	scores := ed.computeScores()

	type scored struct {
		rec   *core.Record
		score float64
	}
	scoredList := make([]scored, 0, len(ed.schemRecs))
	for _, rec := range ed.schemRecs {
		scoredList = append(scoredList, scored{rec: rec, score: scores[rec.Id]})
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

// getCategoryTrendingPage returns a page of trending schematics filtered to a specific category.
func getCategoryTrendingPage(app *pocketbase.PocketBase, cacheService *cache.Service, categoryID string, page int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematics(app, cacheService)
	var filtered []models.Schematic
	for _, s := range all {
		for _, c := range s.Categories {
			if c.ID == categoryID {
				filtered = append(filtered, s)
				break
			}
		}
	}
	offset := (page - 1) * indexPageSize
	if offset >= len(filtered) {
		return nil, false
	}
	end := offset + indexPageSize
	hasNext := end < len(filtered)
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], hasNext
}

// WarmIndexCache pre-computes and caches all data needed for the index page
// so that no user request ever hits the database for page-1 data.
// Called at boot and periodically by a background ticker.
func WarmIndexCache(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) {
	app.Logger().Debug("Warming index page cache")

	// 1. Latest schematics (page 1)
	col, colErr := app.FindCollectionByNameOrId("schematics")
	if colErr == nil {
		latestResults, latestHasNext := fetchLatestPage(app, col.Id, 1)
		cacheService.SetSchematics(cache.LatestSchematicsKey, MapResultsToSchematic(app, latestResults, cacheService))
		cacheService.Set(cache.LatestHasNextKey, latestHasNext)
	}

	// 2. Trending schematics (force recompute by clearing cache first)
	cacheService.Delete(cache.TrendingSchematicsKey)
	allTrending := getAllTrendingSchematics(app, cacheService)
	trendingHasNext := len(allTrending) > indexPageSize
	cacheService.Set(cache.TrendingHasNextKey, trendingHasNext)

	// 3. Highest rated schematics (page 1)
	highestRated, highestHasNext := getHighestRatedSchematicsPage(app, cacheService, 1)
	cacheService.SetSchematics(cache.HighestRatedSchematicsKey, highestRated)
	cacheService.Set(cache.HighestRatedHasNextKey, highestHasNext)

	// 4. Categories (already self-caching, just ensure warm)
	categories := allCategoriesFromStore(appStore, app, cacheService)

	// 5. Category sections — cache page-1 trending items per category
	for _, cat := range categories {
		items, catHasNext := getCategoryTrendingPage(app, cacheService, cat.ID, 1)
		cacheService.SetSchematics(fmt.Sprintf("CategorySection:%s", cat.Key), items)
		cacheService.Set(fmt.Sprintf("CategorySectionHasNext:%s", cat.Key), catHasNext)
	}

	app.Logger().Debug("Index page cache warmed")
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
