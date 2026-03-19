package pages

import (
	"context"
	"createmod/internal/abtest"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/metrics"
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
const indexHTMLCacheTTL = 5 * time.Minute

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

func indexHTMLCacheKey(lang string) string {
	return fmt.Sprintf("IndexHTML:%s", lang)
}

func indexHTMLCacheKeyWithWindow(lang string, windowDays int) string {
	return fmt.Sprintf("IndexHTML:%s:%d", lang, windowDays)
}

// allCategorySectionsPopulated returns true when every category section has at
// least one schematic. An empty section means the data cache was likely cold.
func allCategorySectionsPopulated(sections []CategorySection) bool {
	for _, s := range sections {
		if len(s.Items) == 0 {
			return false
		}
	}
	return true
}

// detectLanguageFromRequest determines the language for the current request
// using the same logic as DefaultData.Populate: X-Createmod-Lang header first,
// then cm_lang cookie, defaulting to "en".
func detectLanguageFromRequest(r *http.Request) string {
	if lang := r.Header.Get("X-Createmod-Lang"); lang != "" && isSupportedLanguage(lang) {
		return lang
	}
	return preferredLanguageFromRequest(r)
}

func IndexHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, trendingConfig *abtest.TrendingConfig) func(e *server.RequestEvent) error {
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

		// Determine trending variant and window
		tv := abtest.TrendingVariantFromContext(e.Request.Context())
		windowDays := 30
		variantName := ""
		if tv != nil && trendingConfig != nil && trendingConfig.Enabled {
			windowDays = tv.WindowDays
			variantName = tv.Name
		}

		// HTMX tab request — return just the tab panel partial
		if isHTMX && tab != "" {
			return renderTabPartial(cacheService, registry, appStore, e, tab, page, windowDays)
		}

		// Record page view metric
		metrics.IndexPageViews.WithLabelValues(variantName, fmt.Sprintf("%d", windowDays)).Inc()

		// For anonymous users, serve from rendered HTML cache (5-minute TTL).
		// Authenticated pages contain user-specific data so are always rendered fresh.
		isAuth := authenticatedUserID(e) != ""
		if !isAuth {
			lang := detectLanguageFromRequest(e.Request)
			htmlCacheKey := indexHTMLCacheKeyWithWindow(lang, windowDays)
			if cached, ok := cacheService.GetString(htmlCacheKey); ok {
				return e.HTML(http.StatusOK, cached)
			}
		}

		// Full page load — serve from pre-warmed cache when available.
		latestSchematics, latestCached := cacheService.GetSchematics(cache.LatestSchematicsKey)

		// Load trending from window-specific cache key
		trendingCacheKey := cache.TrendingKeyForWindow(windowDays)
		trendingHasNextKey := cache.TrendingHasNextKeyForWindow(windowDays)
		trendingSchematics, _ := cacheService.GetSchematics(trendingCacheKey)
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
		if v, ok := cacheService.Get(trendingHasNextKey); ok {
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
			trendingSchematics, trendingHasNext = getTrendingSchematicsPageForWindow(appStore, cacheService, 1, windowDays)
			highestRated, highestHasNext = getHighestRatedSchematicsPageFromStore(appStore, cacheService, 1)
		}

		// Build category sections
		categories := allCategoriesFromStoreOnly(appStore, cacheService)
		categorySections := make([]CategorySection, 0, len(categories))
		for _, cat := range categories {
			cacheKey := cache.CategorySectionKeyForWindow(cat.Key, windowDays)
			cacheHasNextKey := cache.CategorySectionHasNextKeyForWindow(cat.Key, windowDays)
			items, cached := cacheService.GetSchematics(cacheKey)
			catHasNext := false
			if cached {
				if v, ok := cacheService.Get(cacheHasNextKey); ok {
					if b, ok := v.(bool); ok {
						catHasNext = b
					}
				}
			} else {
				items, catHasNext = getCategoryTrendingPageForWindow(appStore, cacheService, cat.ID, 1, windowDays)
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
		d.TrendingVariant = variantName
		d.TrendingWindowDays = windowDays
		d.HideOutstream = true
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

		// Cache the rendered HTML for anonymous users (5-minute TTL).
		// Only cache when all sections have data — if the data caches were
		// cold (e.g. pod just started), the page may have been rendered with
		// empty sections and we don't want to serve that for 5 minutes.
		if !isAuth && len(latestSchematics) > 0 && len(trendingSchematics) > 0 && len(highestRated) > 0 && allCategorySectionsPopulated(categorySections) {
			cacheService.SetWithTTL(indexHTMLCacheKeyWithWindow(d.Language, windowDays), html, indexHTMLCacheTTL)
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

func renderTabPartial(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, e *server.RequestEvent, tab string, page int, windowDays int) error {
	var items []models.Schematic
	var hasNext bool

	switch {
	case tab == "trending":
		items, hasNext = getTrendingSchematicsPageForWindow(appStore, cacheService, page, windowDays)
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
		items, hasNext = getCategoryTrendingPageForWindow(appStore, cacheService, catID, page, windowDays)
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
func WarmIndexCache(cacheService *cache.Service, appStore *store.Store, windowDays []int) {
	// Delegate to the store-based implementation
	WarmIndexCacheFromStore(appStore, cacheService, slog.Default(), windowDays)
}

// RefreshIndexCache asynchronously clears and re-warms the index page cache.
// Call this after a schematic is created, updated, or deleted so the homepage
// reflects the change without waiting for the next periodic job.
func RefreshIndexCache(cacheService *cache.Service, appStore *store.Store, windowDays []int) {
	go func() {
		cacheService.Delete(cache.LatestSchematicsKey)
		cacheService.Delete(cache.LatestHasNextKey)
		cacheService.Delete(cache.TrendingSchematicsKey)
		cacheService.Delete(cache.TrendingHasNextKey)
		cacheService.Delete(cache.HighestRatedSchematicsKey)
		cacheService.Delete(cache.HighestRatedHasNextKey)
		// Clear window-specific keys
		for _, wd := range windowDays {
			cacheService.Delete(cache.TrendingKeyForWindow(wd))
			cacheService.Delete(cache.TrendingHasNextKeyForWindow(wd))
		}
		// Invalidate rendered HTML caches for all languages and windows
		for lang := range supportedLanguages {
			cacheService.Delete(indexHTMLCacheKey(lang))
			for _, wd := range windowDays {
				cacheService.Delete(indexHTMLCacheKeyWithWindow(lang, wd))
			}
		}
		WarmIndexCacheFromStore(appStore, cacheService, slog.Default(), windowDays)
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
// Uses the default 30-day window and default cache key.
func getAllTrendingSchematicsFromStore(appStore *store.Store, cacheService *cache.Service) []models.Schematic {
	return getAllTrendingSchematicsForWindow(appStore, cacheService, 30)
}

// getAllTrendingSchematicsForWindow returns the full sorted trending list for a specific time window.
func getAllTrendingSchematicsForWindow(appStore *store.Store, cacheService *cache.Service, recentDays int) []models.Schematic {
	cacheKey := cache.TrendingKeyForWindow(recentDays)
	cached, found := cacheService.GetSchematics(cacheKey)
	if found {
		return cached
	}

	td, err := appStore.ViewRatings.FetchTrendingData(context.Background(), recentDays)
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
	cacheService.SetSchematics(cacheKey, all)
	return all
}

// getTrendingSchematicsPageFromStore returns a page of trending schematics from the PostgreSQL store (default 30-day window).
func getTrendingSchematicsPageFromStore(appStore *store.Store, cacheService *cache.Service, page int) ([]models.Schematic, bool) {
	return getTrendingSchematicsPageForWindow(appStore, cacheService, page, 30)
}

// getTrendingSchematicsPageForWindow returns a page of trending schematics for a specific time window.
func getTrendingSchematicsPageForWindow(appStore *store.Store, cacheService *cache.Service, page int, recentDays int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematicsForWindow(appStore, cacheService, recentDays)
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

// getCategoryTrendingPageFromStore returns a page of trending schematics filtered to a specific category (default 30-day window).
func getCategoryTrendingPageFromStore(appStore *store.Store, cacheService *cache.Service, categoryID string, page int) ([]models.Schematic, bool) {
	return getCategoryTrendingPageForWindow(appStore, cacheService, categoryID, page, 30)
}

// getCategoryTrendingPageForWindow returns a page of trending schematics filtered to a specific category for a given window.
func getCategoryTrendingPageForWindow(appStore *store.Store, cacheService *cache.Service, categoryID string, page int, recentDays int) ([]models.Schematic, bool) {
	all := getAllTrendingSchematicsForWindow(appStore, cacheService, recentDays)
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
// windowDays specifies which trending time windows to warm; if nil/empty, defaults to [30].
func WarmIndexCacheFromStore(appStore *store.Store, cacheService *cache.Service, logger interface{ Debug(string, ...any) }, windowDays []int) {
	logger.Debug("Warming index page cache (store)")

	if len(windowDays) == 0 {
		windowDays = []int{30}
	}

	// 1. Latest schematics (page 1)
	latestResults, latestHasNext := fetchLatestPageFromStore(appStore, 1)
	cacheService.SetSchematics(cache.LatestSchematicsKey, MapStoreSchematics(appStore, latestResults, cacheService))
	cacheService.Set(cache.LatestHasNextKey, latestHasNext)

	// 2. Trending schematics — warm all requested windows
	for _, wd := range windowDays {
		ck := cache.TrendingKeyForWindow(wd)
		cacheService.Delete(ck)
		allTrending := getAllTrendingSchematicsForWindow(appStore, cacheService, wd)
		trendingHasNext := len(allTrending) > indexPageSize
		cacheService.Set(cache.TrendingHasNextKeyForWindow(wd), trendingHasNext)
	}

	// Backward compat: also populate default (non-windowed) keys with 30-day data
	cacheService.Delete(cache.TrendingSchematicsKey)
	defaultTrending := getAllTrendingSchematicsForWindow(appStore, cacheService, 30)
	cacheService.SetSchematics(cache.TrendingSchematicsKey, defaultTrending)
	cacheService.Set(cache.TrendingHasNextKey, len(defaultTrending) > indexPageSize)

	// 3. Highest rated schematics (page 1)
	highestRated, highestHasNext := getHighestRatedSchematicsPageFromStore(appStore, cacheService, 1)
	cacheService.SetSchematics(cache.HighestRatedSchematicsKey, highestRated)
	cacheService.Set(cache.HighestRatedHasNextKey, highestHasNext)

	// 4. Categories
	allCategoriesFromStoreOnly(appStore, cacheService)

	// 5. Category sections — warm for all requested windows
	categories := allCategoriesFromStoreOnly(appStore, cacheService)
	for _, cat := range categories {
		for _, wd := range windowDays {
			items, catHasNext := getCategoryTrendingPageForWindow(appStore, cacheService, cat.ID, 1, wd)
			cacheService.SetSchematics(cache.CategorySectionKeyForWindow(cat.Key, wd), items)
			cacheService.Set(cache.CategorySectionHasNextKeyForWindow(cat.Key, wd), catHasNext)
		}
		// Backward compat: default keys with 30-day data
		items, catHasNext := getCategoryTrendingPageForWindow(appStore, cacheService, cat.ID, 1, 30)
		cacheService.SetSchematics(fmt.Sprintf("CategorySection:%s", cat.Key), items)
		cacheService.Set(fmt.Sprintf("CategorySectionHasNext:%s", cat.Key), catHasNext)
	}

	logger.Debug("Index page cache warmed (store)")
}

