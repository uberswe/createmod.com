package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/ratelimit"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"time"
)

// homeSegmentSize is how many schematics each home rail returns.
const homeSegmentSize = 12

// apiHomeResponse is the JSON shape for GET /api/home.
type apiHomeResponse struct {
	Trending     []models.Schematic `json:"trending"`
	Latest       []models.Schematic `json:"latest"`
	HighestRated []models.Schematic `json:"highestRated"`
}

// apiFilterCategory is a category/tag option for the search UI.
type apiFilterCategory struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Count int64  `json:"count,omitempty"`
}

// apiFilterCreateVersionGroup mirrors the grouped Create version dropdown.
type apiFilterCreateVersionGroup struct {
	Group    string   `json:"group"`
	Value    string   `json:"value"`
	Versions []string `json:"versions"`
}

// apiFilterMod is a mod option (namespace + display name + count).
type apiFilterMod struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Count     int    `json:"count"`
}

// apiFiltersResponse is the JSON shape for GET /api/schematics/filters.
type apiFiltersResponse struct {
	Categories        []apiFilterCategory           `json:"categories"`
	MinecraftVersions []string                      `json:"minecraftVersions"`
	CreateVersions    []apiFilterCreateVersionGroup `json:"createVersions"`
	Tags              []apiFilterCategory           `json:"tags"`
	Mods              []apiFilterMod                `json:"mods"`
}

// APIHomeHandler serves GET /api/home returning the trending / latest / highest
// rated rails, mirroring the website home page. Auth: API key or HMAC.
func APIHomeHandler(searchEngine search.SearchEngine, rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/home"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, cacheService)
		if err != nil {
			return nil
		}
		if rejected := applyAPIRateLimit(e, rl, keyID, isHMAC); rejected {
			return nil
		}
		if !isHMAC {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
		}

		// The home rails are identical for every caller; cache the assembled
		// response briefly so we don't run the 3x search + hydrate fan-out on
		// every request.
		const cacheKey = "api:home:v1"
		if serveCachedJSON(e, cacheService, cacheKey) {
			return nil
		}

		ctx := context.Background()
		segment := func(order int) []models.Schematic {
			sq := search.SearchQuery{Term: "", Order: order, Rating: -1, Category: "all"}
			ordered, err := apiSearchResults(ctx, searchEngine, appStore, cacheService, sq, homeSegmentSize)
			if err != nil || len(ordered) == 0 {
				return []models.Schematic{}
			}
			if len(ordered) > homeSegmentSize {
				ordered = ordered[:homeSegmentSize]
			}
			items := MapStoreSchematics(appStore, ordered, cacheService)
			for i := range items {
				items[i].SchematicFile = ""
			}
			return items
		}

		resp := apiHomeResponse{
			Trending:     segment(search.TrendingOrder),
			Latest:       segment(search.NewestOrder),
			HighestRated: segment(search.HighestRatingOrder),
		}
		return writeAndCacheJSON(e, cacheService, cacheKey, 60*time.Second, resp)
	}
}

// APIFiltersHandler serves GET /api/schematics/filters returning the option
// lists used to populate the search filter UI. Auth: API key or HMAC.
func APIFiltersHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/filters"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, cacheService)
		if err != nil {
			return nil
		}
		if rejected := applyAPIRateLimit(e, rl, keyID, isHMAC); rejected {
			return nil
		}
		if !isHMAC {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
		}

		// Filter option lists change rarely and are the same for every caller.
		const cacheKey = "api:filters:v1"
		if serveCachedJSON(e, cacheService, cacheKey) {
			return nil
		}

		categories := make([]apiFilterCategory, 0)
		for _, c := range allCategoriesFromStoreOnly(appStore, cacheService) {
			categories = append(categories, apiFilterCategory{Key: c.Key, Name: c.Name})
		}

		mcVersions := make([]string, 0)
		for _, v := range allMinecraftVersionsFromStore(appStore) {
			mcVersions = append(mcVersions, v.Version)
		}

		createVersions := make([]apiFilterCreateVersionGroup, 0)
		for _, g := range groupCreateVersions(allCreatemodVersionsFromStore(appStore)) {
			versions := make([]string, 0, len(g.Versions))
			for _, v := range g.Versions {
				versions = append(versions, v.Version)
			}
			createVersions = append(createVersions, apiFilterCreateVersionGroup{
				Group:    g.Label,
				Value:    g.Value,
				Versions: versions,
			})
		}

		tags := make([]apiFilterCategory, 0)
		for _, t := range allTagsWithCountFromStore(appStore, cacheService) {
			tags = append(tags, apiFilterCategory{Key: t.Key, Name: t.Name, Count: t.Count})
		}

		mods := make([]apiFilterMod, 0)
		for _, m := range allModOptionsFromStore(appStore, cacheService) {
			mods = append(mods, apiFilterMod{Namespace: m.Namespace, Name: m.DisplayName, Count: m.Count})
		}

		resp := apiFiltersResponse{
			Categories:        categories,
			MinecraftVersions: mcVersions,
			CreateVersions:    createVersions,
			Tags:              tags,
			Mods:              mods,
		}
		return writeAndCacheJSON(e, cacheService, cacheKey, 5*time.Minute, resp)
	}
}

// applyAPIRateLimit enforces the standard API rate limit (HMAC: 100/min by IP,
// API key: 120/min). It writes a 429 response and returns true when rejected.
func applyAPIRateLimit(e *server.RequestEvent, rl ratelimit.Limiter, keyID string, isHMAC bool) bool {
	if isHMAC {
		if ok, retry := searchRateLimitAllow(rl, e.RealIP(), 100); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			_ = writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return true
		}
		return false
	}
	if ok, retry := rateLimitAllow(rl, keyID, 120); !ok {
		e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
		_ = writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		return true
	}
	return false
}
