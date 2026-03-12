package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"fmt"
	tmpl "html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"math"
	"sort"
	"time"

	"github.com/drexedam/gravatar"
	"createmod/internal/server"
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

func IndexHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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
			return renderTabPartial(cacheService, registry, appStore, e, tab, page)
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
			latestStoreResults, lhn := fetchLatestPageFromStore(appStore, 1)
			latestSchematics = MapStoreSchematics(appStore, latestStoreResults, cacheService)
			latestHasNext = lhn
			trendingSchematics, trendingHasNext = getTrendingSchematicsPageFromStore(appStore, cacheService, 1)
			highestRated, highestHasNext = getHighestRatedSchematicsPageFromStore(appStore, cacheService, 1)
		}

		// Build category sections
		categories := allCategoriesFromStoreOnly(appStore, cacheService)
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
				items, catHasNext = getCategoryTrendingPageFromStore(appStore, cacheService, cat.ID, 1)
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
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

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

func renderTabPartial(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, e *server.RequestEvent, tab string, page int) error {
	var items []models.Schematic
	var hasNext bool

	switch {
	case tab == "trending":
		items, hasNext = getTrendingSchematicsPageFromStore(appStore, cacheService, page)
	case tab == "highest":
		items, hasNext = getHighestRatedSchematicsPageFromStore(appStore, cacheService, page)
	case strings.HasPrefix(tab, "cat-"):
		catKey := strings.TrimPrefix(tab, "cat-")
		categories := allCategoriesFromStoreOnly(appStore, cacheService)
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
		items, hasNext = getCategoryTrendingPageFromStore(appStore, cacheService, catID, page)
	default:
		tab = "latest"
		storeResults, hn := fetchLatestPageFromStore(appStore, page)
		items = MapStoreSchematics(appStore, storeResults, cacheService)
		hasNext = hn
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

// WarmIndexCache pre-computes and caches all data needed for the index page
// so that no user request ever hits the database for page-1 data.
// Called at boot and periodically by a background ticker.
func WarmIndexCache(cacheService *cache.Service, appStore *store.Store) {
	// Delegate to the store-based implementation
	WarmIndexCacheFromStore(appStore, cacheService, slog.Default())
}

// RefreshIndexCache asynchronously clears and re-warms the index page cache.
// Call this after a schematic is created, updated, or deleted so the homepage
// reflects the change without waiting for the next periodic job.
func RefreshIndexCache(cacheService *cache.Service, appStore *store.Store) {
	go func() {
		cacheService.Delete(cache.LatestSchematicsKey)
		cacheService.Delete(cache.LatestHasNextKey)
		cacheService.Delete(cache.TrendingSchematicsKey)
		cacheService.Delete(cache.TrendingHasNextKey)
		cacheService.Delete(cache.HighestRatedSchematicsKey)
		cacheService.Delete(cache.HighestRatedHasNextKey)
		WarmIndexCacheFromStore(appStore, cacheService, slog.Default())
	}()
}


// ComputeTrendingScoresFromStore computes trending scores using the PostgreSQL store.
// Returns a map of schematic ID to score. Also persists trending scores and
// rating aggregates to the schematics table for pre-computed query support.
func ComputeTrendingScoresFromStore(appStore *store.Store) map[string]float64 {
	ctx := context.Background()
	td, err := appStore.ViewRatings.FetchTrendingData(ctx, 30)
	if err != nil || td == nil {
		return nil
	}
	scores := make(map[string]float64, len(td.SchematicIDs))
	for _, id := range td.SchematicIDs {
		created := td.SchematicCreated[id]
		scores[id] = trendingScore(created, td.RecentViews[id], td.TotalViews[id], td.RatingCount[id], td.RatingSum[id], td.RecentDownloads[id], td.TotalDownloads[id])
	}

	// Persist trending scores and rating aggregates to the schematics table
	for _, id := range td.SchematicIDs {
		if err := appStore.Schematics.UpdateTrendingScore(ctx, id, scores[id]); err != nil {
			slog.Error("failed to persist trending score", "schematicID", id, "error", err)
		}

		rSum := td.RatingSum[id]
		rCount := td.RatingCount[id]
		if rCount > 0 {
			avgRating := rSum / rCount
			if err := appStore.Schematics.UpdateRatingAggregates(ctx, id, avgRating, int(rCount)); err != nil {
				slog.Error("failed to persist rating aggregates", "schematicID", id, "error", err)
			}
		}
	}

	return scores
}

// fetchLatestPageFromStore fetches a page of latest approved schematics from the PostgreSQL store.
func fetchLatestPageFromStore(appStore *store.Store, page int) ([]store.Schematic, bool) {
	limit := indexPageSize + 1
	offset := (page - 1) * indexPageSize
	results, err := appStore.Schematics.ListApproved(context.Background(), limit, offset)
	if err != nil {
		return nil, false
	}
	hasNext := len(results) > indexPageSize
	if hasNext {
		results = results[:indexPageSize]
	}
	return results, hasNext
}

// getHighestRatedSchematicsPageFromStore fetches a page of highest rated schematics from the PostgreSQL store.
func getHighestRatedSchematicsPageFromStore(appStore *store.Store, cacheService *cache.Service, page int) ([]models.Schematic, bool) {
	limit := indexPageSize + 1
	offset := (page - 1) * indexPageSize
	results, err := appStore.Schematics.ListHighestRated(context.Background(), limit, offset)
	if err != nil {
		return nil, false
	}
	hasNext := len(results) > indexPageSize
	if hasNext {
		results = results[:indexPageSize]
	}
	return MapStoreSchematics(appStore, results, cacheService), hasNext
}

// getAllTrendingSchematicsFromStore returns the full sorted trending list using the PostgreSQL store.
func getAllTrendingSchematicsFromStore(appStore *store.Store, cacheService *cache.Service) []models.Schematic {
	cached, found := cacheService.GetSchematics(cache.TrendingSchematicsKey)
	if found {
		return cached
	}

	td, err := appStore.ViewRatings.FetchTrendingData(context.Background(), 30)
	if err != nil || td == nil || len(td.SchematicIDs) == 0 {
		return nil
	}

	// Compute scores
	type scored struct {
		id    string
		score float64
	}
	scoredList := make([]scored, 0, len(td.SchematicIDs))
	for _, id := range td.SchematicIDs {
		created := td.SchematicCreated[id]
		s := trendingScore(created, td.RecentViews[id], td.TotalViews[id], td.RatingCount[id], td.RatingSum[id], td.RecentDownloads[id], td.TotalDownloads[id])
		scoredList = append(scoredList, scored{id: id, score: s})
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].score > scoredList[j].score })

	// Fetch full schematics in sorted order
	ids := make([]string, len(scoredList))
	for i, s := range scoredList {
		ids[i] = s.id
	}
	storeSchematics, err := appStore.Schematics.ListByIDs(context.Background(), ids)
	if err != nil {
		return nil
	}
	// ListByIDs does not preserve order; re-sort
	schematicMap := make(map[string]store.Schematic, len(storeSchematics))
	for _, s := range storeSchematics {
		schematicMap[s.ID] = s
	}
	ordered := make([]store.Schematic, 0, len(ids))
	for _, id := range ids {
		if s, ok := schematicMap[id]; ok {
			ordered = append(ordered, s)
		}
	}

	all := MapStoreSchematics(appStore, ordered, cacheService)
	cacheService.SetSchematics(cache.TrendingSchematicsKey, all)
	return all
}

// getTrendingSchematicsPageFromStore returns a page of trending schematics from the PostgreSQL store.
func getTrendingSchematicsPageFromStore(appStore *store.Store, cacheService *cache.Service, page int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematicsFromStore(appStore, cacheService)
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

// getCategoryTrendingPageFromStore returns a page of trending schematics filtered to a specific category from the PostgreSQL store.
func getCategoryTrendingPageFromStore(appStore *store.Store, cacheService *cache.Service, categoryID string, page int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematicsFromStore(appStore, cacheService)
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

// findUserFromStore looks up a user by ID from the PostgreSQL store and returns a models.User.
func findUserFromStore(appStore *store.Store, userID string) *models.User {
	if userID == "" {
		return nil
	}
	u, err := appStore.Users.GetUserByID(context.Background(), userID)
	if err != nil || u == nil {
		return nil
	}
	caser := cases.Title(language.English)
	avatarUrl := u.Avatar
	if avatarUrl == "" {
		avatarUrl = gravatar.New(u.Email).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
	}
	return &models.User{
		ID:        u.ID,
		Username:  caser.String(u.Username),
		Avatar:    tmpl.URL(avatarUrl),
		HasAvatar: len(avatarUrl) > 0,
	}
}

// WarmIndexCacheFromStore pre-computes and caches all data needed for the index page
// using only the PostgreSQL store (no PocketBase dependency).
func WarmIndexCacheFromStore(appStore *store.Store, cacheService *cache.Service, logger interface{ Debug(string, ...any) }) {
	logger.Debug("Warming index page cache (store)")

	// 1. Latest schematics (page 1)
	latestResults, latestHasNext := fetchLatestPageFromStore(appStore, 1)
	cacheService.SetSchematics(cache.LatestSchematicsKey, MapStoreSchematics(appStore, latestResults, cacheService))
	cacheService.Set(cache.LatestHasNextKey, latestHasNext)

	// 2. Trending schematics
	cacheService.Delete(cache.TrendingSchematicsKey)
	allTrending := getAllTrendingSchematicsFromStore(appStore, cacheService)
	trendingHasNext := len(allTrending) > indexPageSize
	cacheService.Set(cache.TrendingHasNextKey, trendingHasNext)

	// 3. Highest rated schematics (page 1)
	highestRated, highestHasNext := getHighestRatedSchematicsPageFromStore(appStore, cacheService, 1)
	cacheService.SetSchematics(cache.HighestRatedSchematicsKey, highestRated)
	cacheService.Set(cache.HighestRatedHasNextKey, highestHasNext)

	// 4. Categories
	allCategoriesFromStoreOnly(appStore, cacheService)

	// 5. Category sections
	categories := allCategoriesFromStoreOnly(appStore, cacheService)
	for _, cat := range categories {
		items, catHasNext := getCategoryTrendingPageFromStore(appStore, cacheService, cat.ID, 1)
		cacheService.SetSchematics(fmt.Sprintf("CategorySection:%s", cat.Key), items)
		cacheService.Set(fmt.Sprintf("CategorySectionHasNext:%s", cat.Key), catHasNext)
	}

	logger.Debug("Index page cache warmed (store)")
}

