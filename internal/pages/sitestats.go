package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	html "html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"createmod/internal/server"
	"sync"
)

var siteStatsTemplates = append([]string{
	"./template/site-stats.html",
}, commonTemplates...)

type SiteSearchEntry struct {
	Query       string
	SearchCount int
}

type TopViewedEntry struct {
	ID            string
	Name          string
	Title         string
	FeaturedImage string
	TotalViews    int64
}

// CachedSearchStats holds all search-related stats for a given time window.
// Stored under a single cache key per window so the background warming job
// can populate everything in one pass.
type CachedSearchStats struct {
	TopSearches      []SiteSearchEntry
	TopSchematics    []TopViewedEntry
	SearchVolumeJSON string
	TrendingJSON     string
}

type SiteStatsData struct {
	DefaultData

	HourlyViewsJSON     html.JS
	HourlyDownloadsJSON html.JS

	TotalViews30d     int
	TotalDownloads30d int
	GlobalVDRatio     string

	TotalSchematics int64
	TotalDrafts     int64
	DailyUploadsJSON html.JS

	ShowYourStatsLink bool

	CanViewSearchStats bool
	SearchStatsNotice  string

	TopSearches   []SiteSearchEntry
	TopSchematics []TopViewedEntry

	SearchVolumeJSON        html.JS
	TrendingSearchTermsJSON html.JS

	ActiveWindow string
}

func SiteStatsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := SiteStatsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Site Statistics")
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Site Statistics"))
		d.Slug = "/stats"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := context.Background()
		now := time.Now().UTC()
		since30d := now.AddDate(0, 0, -30)

		window := e.Request.URL.Query().Get("window")
		if window != "7d" {
			window = "30d"
		}
		d.ActiveWindow = window

		type cachedGlobal struct {
			Views, Downloads       string
			TotalViews, TotalDL    int
			VDRatio                string
		}

		cacheKey := "site_stats_global"
		var global cachedGlobal
		if cached, found := cacheService.Get(cacheKey); found {
			if g, ok := cached.(cachedGlobal); ok {
				global = g
			}
		}

		if global.Views == "" {
			var hViews, hDownloads []store.HourlyStat
			var ratio float64
			var wg sync.WaitGroup
			wg.Add(3)
			go func() { defer wg.Done(); hViews, _ = appStore.Stats.HourlyStats(ctx, "schematic_views", since30d) }()
			go func() { defer wg.Done(); hDownloads, _ = appStore.Stats.HourlyStats(ctx, "schematic_downloads", since30d) }()
			go func() {
				defer wg.Done()
				if cached, ok := cacheService.GetFloat("site_avg_vd_ratio_v2"); ok {
					ratio = cached
				} else {
					ratio, _ = appStore.Stats.GetSiteAvgVDRatioSinceCutoff(ctx, HourlyTrackingCutoff)
					cacheService.SetFloat("site_avg_vd_ratio_v2", ratio)
				}
			}()
			wg.Wait()

			var tv, td int
			for _, v := range hViews {
				tv += int(v.Count)
			}
			for _, dl := range hDownloads {
				td += int(dl.Count)
			}

			global = cachedGlobal{
				Views:      hourlyStatsJSON(hViews),
				Downloads:  hourlyStatsJSON(hDownloads),
				TotalViews: tv,
				TotalDL:    td,
				VDRatio:    fmt.Sprintf("%.2f%%", ratio*100),
			}
			cacheService.SetWithTTL(cacheKey, global, 30*time.Minute)
		}

		d.HourlyViewsJSON = html.JS(global.Views)
		d.HourlyDownloadsJSON = html.JS(global.Downloads)
		d.TotalViews30d = global.TotalViews
		d.TotalDownloads30d = global.TotalDL
		d.GlobalVDRatio = global.VDRatio

		type cachedCounts struct {
			TotalSchematics int64
			TotalDrafts     int64
			DailyUploads    string
		}
		countsCacheKey := "site_stats_counts"
		var counts cachedCounts
		if cached, found := cacheService.Get(countsCacheKey); found {
			if c, ok := cached.(cachedCounts); ok {
				counts = c
			}
		}
		if counts.DailyUploads == "" {
			var daily []store.DailyCount
			var wgC sync.WaitGroup
			wgC.Add(3)
			go func() { defer wgC.Done(); counts.TotalSchematics, _ = appStore.Schematics.CountApproved(ctx) }()
			go func() { defer wgC.Done(); counts.TotalDrafts, _ = appStore.TempUploads.CountAll(ctx) }()
			go func() { defer wgC.Done(); daily, _ = appStore.Stats.DailySchematicUploads(ctx, since30d) }()
			wgC.Wait()
			counts.DailyUploads = dailyCountsJSON(daily)
			cacheService.SetWithTTL(countsCacheKey, counts, 30*time.Minute)
		}
		d.TotalSchematics = counts.TotalSchematics
		d.TotalDrafts = counts.TotalDrafts
		d.DailyUploadsJSON = html.JS(counts.DailyUploads)

		userID := authenticatedUserID(e)
		d.ShowYourStatsLink = userID != ""

		if userID != "" {
			if isSuperAdmin(e) {
				d.CanViewSearchStats = true
			} else {
				recentUploadKey := fmt.Sprintf("site_stats_recent_upload_%s", userID)
				hasRecent := false
				if cached, found := cacheService.Get(recentUploadKey); found {
					if b, ok := cached.(bool); ok {
						hasRecent = b
					}
				} else {
					cutoff := now.AddDate(0, -3, 0)
					hasRecent, _ = appStore.SearchTracking.HasRecentApprovedUpload(ctx, userID, cutoff)
					cacheService.SetWithTTL(recentUploadKey, hasRecent, 30*time.Minute)
				}
				d.CanViewSearchStats = hasRecent
				if !hasRecent {
					d.SearchStatsNotice = i18n.T(d.Language, "Search and page statistics are available to creators with at least one approved upload in the last 3 months.")
				}
			}
		} else {
			d.SearchStatsNotice = i18n.T(d.Language, "Log in to view search and page statistics. This feature is available to creators with at least one approved upload in the last 3 months.")
		}

		if d.CanViewSearchStats {
			stats := WarmSearchStatsCache(ctx, cacheService, appStore, window)
			d.TopSearches = stats.TopSearches
			d.TopSchematics = stats.TopSchematics
			d.SearchVolumeJSON = html.JS(stats.SearchVolumeJSON)
			d.TrendingSearchTermsJSON = html.JS(stats.TrendingJSON)
		}

		out, err := registry.LoadFiles(siteStatsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, out)
	}
}

func dailyCountsJSON(counts []store.DailyCount) string {
	type point struct {
		X string `json:"x"`
		Y int64  `json:"y"`
	}
	pts := make([]point, len(counts))
	for i, c := range counts {
		pts[i] = point{X: c.Day, Y: c.Count}
	}
	b, _ := json.Marshal(pts)
	return string(b)
}

func searchTermSeriesJSON(terms []string, data []store.SearchTermDailyCount) string {
	type point struct {
		X string `json:"x"`
		Y int64  `json:"y"`
	}
	type series struct {
		Name string  `json:"name"`
		Data []point `json:"data"`
	}

	byTerm := make(map[string][]point, len(terms))
	for _, d := range data {
		byTerm[d.Query] = append(byTerm[d.Query], point{X: d.Day, Y: d.Count})
	}

	result := make([]series, 0, len(terms))
	for _, t := range terms {
		if pts, ok := byTerm[t]; ok {
			result = append(result, series{Name: t, Data: pts})
		}
	}

	b, _ := json.Marshal(result)
	return string(b)
}

// SearchStatsCacheKey returns the cache key for a given time window.
func SearchStatsCacheKey(window string) string {
	return fmt.Sprintf("site_stats_search_all_%s", window)
}

// WarmSearchStatsCache computes search stats for the given window, using the
// cache if available. On cache miss it runs all queries in parallel and stores
// the result for 1 hour. Called by both the HTTP handler and the background
// warming job.
func WarmSearchStatsCache(ctx context.Context, cacheService *cache.Service, appStore *store.Store, window string) CachedSearchStats {
	cacheKey := SearchStatsCacheKey(window)
	if cached, found := cacheService.Get(cacheKey); found {
		if stats, ok := cached.(CachedSearchStats); ok {
			return stats
		}
	}

	now := time.Now().UTC()
	var windowSince time.Time
	if window == "7d" {
		windowSince = now.AddDate(0, 0, -7)
	} else {
		windowSince = now.AddDate(0, 0, -30)
	}

	var topSearches []SiteSearchEntry
	var topSchematics []TopViewedEntry
	var volJSON string

	var wg sync.WaitGroup
	wg.Add(3)

	// Group A: top searches (sequential chain internally)
	go func() {
		defer wg.Done()
		raw, _ := appStore.SearchTracking.ListTopSearchesSince(ctx, windowSince, 200)
		if len(raw) == 0 {
			return
		}
		terms := make([]string, len(raw))
		for i, r := range raw {
			terms[i] = r.Query
		}
		dirty, _ := appStore.SearchTracking.ListDirtySearchTerms(ctx, terms)
		dirtySet := make(map[string]bool, len(dirty))
		for _, d := range dirty {
			dirtySet[d] = true
		}
		var candidates []SiteSearchEntry
		for _, r := range raw {
			if !dirtySet[r.Query] && len(r.Query) >= 3 {
				candidates = append(candidates, SiteSearchEntry{
					Query:       r.Query,
					SearchCount: r.ResultsCount,
				})
			}
		}
		candidates = filterPrefixQueries(candidates)
		if len(candidates) > 100 {
			candidates = candidates[:100]
		}
		topSearches = candidates
	}()

	// Group B: top viewed schematics (independent)
	go func() {
		defer wg.Done()
		topSchem, _ := appStore.SearchTracking.ListTopViewedSchematicsSince(ctx, windowSince, 100)
		entries := make([]TopViewedEntry, len(topSchem))
		for i, s := range topSchem {
			entries[i] = TopViewedEntry{
				ID:            s.ID,
				Name:          s.Name,
				Title:         s.Title,
				FeaturedImage: s.FeaturedImage,
				TotalViews:    s.TotalViews,
			}
		}
		topSchematics = entries
	}()

	// Group C: daily search volume (independent)
	go func() {
		defer wg.Done()
		vol, _ := appStore.SearchTracking.DailySearchVolume(ctx, windowSince)
		volJSON = dailyCountsJSON(vol)
	}()

	wg.Wait()

	// Trending term volume depends on topSearches, so it runs after the WaitGroup.
	var trendJSON string
	if len(topSearches) > 0 {
		limit := 10
		if len(topSearches) < limit {
			limit = len(topSearches)
		}
		terms := make([]string, limit)
		for i := 0; i < limit; i++ {
			terms[i] = topSearches[i].Query
		}
		termVol, _ := appStore.SearchTracking.DailySearchTermVolume(ctx, windowSince, terms)
		trendJSON = searchTermSeriesJSON(terms, termVol)
	} else {
		trendJSON = "[]"
	}

	result := CachedSearchStats{
		TopSearches:      topSearches,
		TopSchematics:    topSchematics,
		SearchVolumeJSON: volJSON,
		TrendingJSON:     trendJSON,
	}
	cacheService.SetWithTTL(cacheKey, result, 1*time.Hour)
	return result
}

// filterPrefixQueries removes queries that are likely typing prefixes of a
// more popular query. A short query is considered a prefix if a longer query
// starting with the same characters has an equal or higher search count.
func filterPrefixQueries(entries []SiteSearchEntry) []SiteSearchEntry {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SearchCount > entries[j].SearchCount
	})

	kept := make([]SiteSearchEntry, 0, len(entries))
	for _, e := range entries {
		lower := strings.ToLower(e.Query)
		isPrefix := false
		for _, k := range kept {
			if strings.HasPrefix(strings.ToLower(k.Query), lower) && len(k.Query) > len(e.Query) {
				isPrefix = true
				break
			}
		}
		if !isPrefix {
			kept = append(kept, e)
		}
	}
	return kept
}
