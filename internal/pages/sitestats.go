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
	"time"

	"createmod/internal/server"
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

type SiteStatsData struct {
	DefaultData

	HourlyViewsJSON     html.JS
	HourlyDownloadsJSON html.JS

	TotalViews30d    int
	TotalDownloads30d int
	GlobalVDRatio    string

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

		var windowSince time.Time
		if window == "7d" {
			windowSince = now.AddDate(0, 0, -7)
		} else {
			windowSince = since30d
		}

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
			hViews, _ := appStore.Stats.HourlyStats(ctx, "schematic_views", since30d)
			hDownloads, _ := appStore.Stats.HourlyStats(ctx, "schematic_downloads", since30d)

			var tv, td int
			for _, v := range hViews {
				tv += int(v.Count)
			}
			for _, dl := range hDownloads {
				td += int(dl.Count)
			}

			var ratio float64
			if cached, ok := cacheService.GetFloat("site_avg_vd_ratio_v2"); ok {
				ratio = cached
			} else {
				ratio, _ = appStore.Stats.GetSiteAvgVDRatioSinceCutoff(ctx, HourlyTrackingCutoff)
				cacheService.SetFloat("site_avg_vd_ratio_v2", ratio)
			}

			global = cachedGlobal{
				Views:      hourlyStatsJSON(hViews),
				Downloads:  hourlyStatsJSON(hDownloads),
				TotalViews: tv,
				TotalDL:    td,
				VDRatio:    fmt.Sprintf("%.2f%%", ratio*100),
			}
			cacheService.SetWithTTL(cacheKey, global, 15*time.Minute)
		}

		d.HourlyViewsJSON = html.JS(global.Views)
		d.HourlyDownloadsJSON = html.JS(global.Downloads)
		d.TotalViews30d = global.TotalViews
		d.TotalDownloads30d = global.TotalDL
		d.GlobalVDRatio = global.VDRatio

		userID := authenticatedUserID(e)
		d.ShowYourStatsLink = userID != ""

		if userID != "" {
			cutoff := now.AddDate(0, -3, 0)
			hasRecent, _ := appStore.SearchTracking.HasRecentApprovedUpload(ctx, userID, cutoff)
			d.CanViewSearchStats = hasRecent
			if !hasRecent {
				d.SearchStatsNotice = i18n.T(d.Language, "Search and page statistics are available to creators with at least one approved upload in the last 3 months.")
			}
		} else {
			d.SearchStatsNotice = i18n.T(d.Language, "Log in to view search and page statistics. This feature is available to creators with at least one approved upload in the last 3 months.")
		}

		if d.CanViewSearchStats {
			searchCacheKey := fmt.Sprintf("site_stats_searches_%s", window)
			schemCacheKey := fmt.Sprintf("site_stats_schematics_%s", window)

			if cached, found := cacheService.Get(searchCacheKey); found {
				if entries, ok := cached.([]SiteSearchEntry); ok {
					d.TopSearches = entries
				}
			}
			if cached, found := cacheService.Get(schemCacheKey); found {
				if entries, ok := cached.([]TopViewedEntry); ok {
					d.TopSchematics = entries
				}
			}

			if d.TopSearches == nil {
				raw, _ := appStore.SearchTracking.ListTopSearchesSince(ctx, windowSince, 200)
				if len(raw) > 0 {
					terms := make([]string, len(raw))
					for i, r := range raw {
						terms[i] = r.Query
					}
					clean, _ := appStore.SearchTracking.ListCleanSearchTerms(ctx, terms)
					cleanSet := make(map[string]bool, len(clean))
					for _, c := range clean {
						cleanSet[c] = true
					}
					for _, r := range raw {
						if cleanSet[r.Query] && len(d.TopSearches) < 100 {
							d.TopSearches = append(d.TopSearches, SiteSearchEntry{
								Query:       r.Query,
								SearchCount: r.ResultsCount,
							})
						}
					}
				}
				cacheService.SetWithTTL(searchCacheKey, d.TopSearches, 1*time.Hour)
			}

			if d.TopSchematics == nil {
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
				d.TopSchematics = entries
				cacheService.SetWithTTL(schemCacheKey, d.TopSchematics, 1*time.Hour)
			}

			volCacheKey := fmt.Sprintf("site_stats_search_vol_%s", window)
			if cached, found := cacheService.Get(volCacheKey); found {
				if js, ok := cached.(string); ok {
					d.SearchVolumeJSON = html.JS(js)
				}
			}
			if d.SearchVolumeJSON == "" {
				vol, _ := appStore.SearchTracking.DailySearchVolume(ctx, windowSince)
				d.SearchVolumeJSON = html.JS(dailyCountsJSON(vol))
				cacheService.SetWithTTL(volCacheKey, string(d.SearchVolumeJSON), 1*time.Hour)
			}

			trendCacheKey := fmt.Sprintf("site_stats_trending_%s", window)
			if cached, found := cacheService.Get(trendCacheKey); found {
				if js, ok := cached.(string); ok {
					d.TrendingSearchTermsJSON = html.JS(js)
				}
			}
			if d.TrendingSearchTermsJSON == "" {
				if len(d.TopSearches) > 0 {
					limit := 10
					if len(d.TopSearches) < limit {
						limit = len(d.TopSearches)
					}
					terms := make([]string, limit)
					for i := 0; i < limit; i++ {
						terms[i] = d.TopSearches[i].Query
					}
					termVol, _ := appStore.SearchTracking.DailySearchTermVolume(ctx, windowSince, terms)
					d.TrendingSearchTermsJSON = html.JS(searchTermSeriesJSON(terms, termVol))
				} else {
					d.TrendingSearchTermsJSON = html.JS("[]")
				}
				cacheService.SetWithTTL(trendCacheKey, string(d.TrendingSearchTermsJSON), 1*time.Hour)
			}
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
