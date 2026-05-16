package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"fmt"
	html "html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"createmod/internal/server"
)

var userStatsTemplates = append([]string{
	"./template/user-statistics.html",
}, commonTemplates...)

type MonthlyDataPoint struct {
	Period string
	Label  string
	Total  int
}

type SchematicStatsRow struct {
	ID            string
	Name          string
	Title         string
	FeaturedImage string
	Views         int
	Downloads     int
	VDRatio       float64
	VDRatioPercent string
	VDRatioBetter bool
	Created       time.Time
}

type UserStatsData struct {
	DefaultData
	MonthlyViews       []MonthlyDataPoint
	MonthlyDownloads   []MonthlyDataPoint
	ViewLabelsJSON     html.JS
	ViewValuesJSON     html.JS
	DownloadLabelsJSON html.JS
	DownloadValuesJSON html.JS

	HourlyViewsJSON      html.JS
	HourlyDownloadsJSON  html.JS
	HourlyVideoPlaysJSON html.JS
	HourlyYTClicksJSON   html.JS
	HourlyTimeOnPageJSON html.JS
	HourlyLayerViewsJSON html.JS

	Schematics       []SchematicStatsRow
	TotalSchematics  int
	Page             int
	PageSize         int
	TotalPages       int
	SiteAvgVDRatio   float64

	TotalViews30d    int
	TotalDownloads30d int
	VDRatioPercent   string
	VDRatioBetter    bool

	HasPrevPage bool
	HasNextPage bool
	PrevPage    int
	NextPage    int

	ShowNewFeatureBanner bool
}

func UserStatsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)

		d := UserStatsData{}
		d.Populate(e)
		d.SettingsPage = "statistics"
		d.HideOutstream = true
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Statistics"))
		d.Title = i18n.T(d.Language, "Statistics")
		d.Description = i18n.T(d.Language, "page.userstats.description")
		d.Slug = "/settings/statistics"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := context.Background()
		now := time.Now().UTC()
		start := now.AddDate(0, -11, 0)
		since := now.AddDate(0, 0, -30)

		// --- Cached monthly stats (30 min TTL) ---
		type cachedMonthly struct {
			ViewLabels, ViewValues, DLLabels, DLValues string
			Views, Downloads                           []MonthlyDataPoint
		}
		monthlyCacheKey := fmt.Sprintf("user_monthly_stats:%s", userID)
		var monthly cachedMonthly
		if cached, found := cacheService.Get(monthlyCacheKey); found {
			if cm, ok := cached.(cachedMonthly); ok {
				monthly = cm
			}
		}
		needMonthly := monthly.ViewLabels == ""

		// --- Cached hourly stats (15 min TTL) ---
		hourlyCacheKey := fmt.Sprintf("user_analytics:%s", userID)
		type cachedHourly struct {
			Views, Downloads, VideoPlays, YTClicks, TimeOnPage, LayerViews string
			TotalViews, TotalDownloads                                     int
		}
		var hourly cachedHourly
		if cached, found := cacheService.Get(hourlyCacheKey); found {
			if ch, ok := cached.(cachedHourly); ok {
				hourly = ch
			}
		}
		needHourly := hourly.Views == ""

		// --- Site VD ratio (shared cache) ---
		var siteAvg float64
		siteAvgCached := false
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio_v2"); ok {
			siteAvg = cached
			siteAvgCached = true
		}

		// --- Run all uncached queries in parallel ---
		var wg sync.WaitGroup
		if needMonthly {
			wg.Add(1)
			go func() {
				defer wg.Done()
				monthlyStats, err := appStore.Stats.MonthlyUserStats(ctx, userID, 12)
				var views, downloads []MonthlyDataPoint
				if err == nil {
					for _, s := range monthlyStats {
						views = append(views, MonthlyDataPoint{
							Period: s.Month,
							Label:  periodToLabel(s.Month),
							Total:  int(s.Views),
						})
						downloads = append(downloads, MonthlyDataPoint{
							Period: s.Month,
							Label:  periodToLabel(s.Month),
							Total:  int(s.Downloads),
						})
					}
				}
				filled := fillMissingMonths(views, start, now)
				filledDL := fillMissingMonths(downloads, start, now)
				monthly = cachedMonthly{
					ViewLabels: toJSONStringArray(filled),
					ViewValues: toJSONIntArray(filled),
					DLLabels:   toJSONStringArray(filledDL),
					DLValues:   toJSONIntArray(filledDL),
					Views:      filled,
					Downloads:  filledDL,
				}
				cacheService.SetWithTTL(monthlyCacheKey, monthly, 30*time.Minute)
			}()
		}
		if needHourly {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var hViews, hDownloads, hVideoPlays, hYTClicks, hTimeOnPage, hLayerViews []store.HourlyStat
				var hwg sync.WaitGroup
				hwg.Add(6)
				go func() { defer hwg.Done(); hViews, _ = appStore.Stats.HourlyUserViews(ctx, userID, since) }()
				go func() { defer hwg.Done(); hDownloads, _ = appStore.Stats.HourlyUserDownloads(ctx, userID, since) }()
				go func() { defer hwg.Done(); hVideoPlays, _ = appStore.Stats.HourlyUserEvents(ctx, userID, store.EventVideoPlay, since) }()
				go func() { defer hwg.Done(); hYTClicks, _ = appStore.Stats.HourlyUserEvents(ctx, userID, store.EventYouTubeClick, since) }()
				go func() { defer hwg.Done(); hTimeOnPage, _ = appStore.Stats.HourlyUserEventAvg(ctx, userID, store.EventTimeOnPage, since) }()
				go func() { defer hwg.Done(); hLayerViews, _ = appStore.Stats.HourlyUserEvents(ctx, userID, store.EventLayerViewer, since) }()
				hwg.Wait()

				var tv, td int
				for _, v := range hViews {
					tv += int(v.Count)
				}
				for _, dl := range hDownloads {
					td += int(dl.Count)
				}
				hourly = cachedHourly{
					Views:          hourlyStatsJSON(hViews),
					Downloads:      hourlyStatsJSON(hDownloads),
					VideoPlays:     hourlyStatsJSON(hVideoPlays),
					YTClicks:       hourlyStatsJSON(hYTClicks),
					TimeOnPage:     hourlyStatsJSON(hTimeOnPage),
					LayerViews:     hourlyStatsJSON(hLayerViews),
					TotalViews:     tv,
					TotalDownloads: td,
				}
				cacheService.SetWithTTL(hourlyCacheKey, hourly, 15*time.Minute)
			}()
		}
		if !siteAvgCached {
			wg.Add(1)
			go func() {
				defer wg.Done()
				siteAvg, _ = appStore.Stats.GetSiteAvgVDRatioSinceCutoff(ctx, HourlyTrackingCutoff)
				cacheService.SetFloat("site_avg_vd_ratio_v2", siteAvg)
			}()
		}
		wg.Wait()

		d.MonthlyViews = monthly.Views
		d.MonthlyDownloads = monthly.Downloads
		d.ViewLabelsJSON = html.JS(monthly.ViewLabels)
		d.ViewValuesJSON = html.JS(monthly.ViewValues)
		d.DownloadLabelsJSON = html.JS(monthly.DLLabels)
		d.DownloadValuesJSON = html.JS(monthly.DLValues)

		d.HourlyViewsJSON = html.JS(hourly.Views)
		d.HourlyDownloadsJSON = html.JS(hourly.Downloads)
		d.HourlyVideoPlaysJSON = html.JS(hourly.VideoPlays)
		d.HourlyYTClicksJSON = html.JS(hourly.YTClicks)
		d.HourlyTimeOnPageJSON = html.JS(hourly.TimeOnPage)
		d.HourlyLayerViewsJSON = html.JS(hourly.LayerViews)
		d.TotalViews30d = hourly.TotalViews
		d.TotalDownloads30d = hourly.TotalDownloads

		d.SiteAvgVDRatio = siteAvg
		var vdRatio float64
		if d.TotalViews30d > 0 {
			vdRatio = float64(d.TotalDownloads30d) / float64(d.TotalViews30d)
		}
		d.VDRatioPercent = fmt.Sprintf("%.2f%%", vdRatio*100)
		d.VDRatioBetter = vdRatio >= siteAvg

		// --- Schematic list with pagination (cached 15 min) ---
		d.PageSize = 20
		pageStr := e.Request.URL.Query().Get("page")
		d.Page, _ = strconv.Atoi(pageStr)
		if d.Page < 1 {
			d.Page = 1
		}
		offset := (d.Page - 1) * d.PageSize

		type cachedSchematicList struct {
			Total int
			Stats []store.SchematicStatsSummary
		}
		schemCacheKey := fmt.Sprintf("user_schem_stats:%s:%d", userID, d.Page)
		var schemList cachedSchematicList
		if cached, found := cacheService.Get(schemCacheKey); found {
			if cl, ok := cached.(cachedSchematicList); ok {
				schemList = cl
			}
		}
		if schemList.Stats == nil {
			var totalSchematics int
			var schematicStats []store.SchematicStatsSummary
			var slWg sync.WaitGroup
			slWg.Add(2)
			go func() { defer slWg.Done(); totalSchematics, _ = appStore.Stats.CountUserSchematics(ctx, userID) }()
			go func() { defer slWg.Done(); schematicStats, _ = appStore.Stats.ListSchematicStats(ctx, userID, d.PageSize, offset) }()
			slWg.Wait()
			schemList = cachedSchematicList{Total: totalSchematics, Stats: schematicStats}
			if schemList.Stats == nil {
				schemList.Stats = []store.SchematicStatsSummary{}
			}
			cacheService.SetWithTTL(schemCacheKey, schemList, 15*time.Minute)
		}

		d.TotalSchematics = schemList.Total
		d.TotalPages = (schemList.Total + d.PageSize - 1) / d.PageSize
		d.HasPrevPage = d.Page > 1
		d.HasNextPage = d.Page < d.TotalPages
		d.PrevPage = d.Page - 1
		d.NextPage = d.Page + 1

		cutoff := HourlyTrackingCutoff
		for _, s := range schemList.Stats {
			var ratio float64
			if s.Views > 0 {
				ratio = float64(s.Downloads) / float64(s.Views)
			}
			d.Schematics = append(d.Schematics, SchematicStatsRow{
				ID:             s.ID,
				Name:           s.Name,
				Title:          s.Title,
				FeaturedImage:  s.FeaturedImage,
				Views:          s.Views,
				Downloads:      s.Downloads,
				VDRatio:        ratio,
				VDRatioPercent: fmt.Sprintf("%.2f%%", ratio*100),
				VDRatioBetter:  ratio >= siteAvg,
				Created:        s.Created,
			})
			if s.Created.Before(cutoff) {
				d.ShowNewFeatureBanner = true
			}
		}

		out, err := registry.LoadFiles(userStatsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, out)
	}
}

func periodToLabel(period string) string {
	if len(period) < 6 {
		return period
	}
	year, err := strconv.Atoi(period[:4])
	if err != nil {
		return period
	}
	month, err := strconv.Atoi(period[4:6])
	if err != nil || month < 1 || month > 12 {
		return period
	}
	return fmt.Sprintf("%s %d", time.Month(month).String()[:3], year)
}

func fillMissingMonths(data []MonthlyDataPoint, start, end time.Time) []MonthlyDataPoint {
	lookup := make(map[string]int, len(data))
	for _, dp := range data {
		lookup[dp.Period] = dp.Total
	}

	var result []MonthlyDataPoint
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(endMonth) {
		period := fmt.Sprintf("%d%02d", cur.Year(), int(cur.Month()))
		total := lookup[period]
		result = append(result, MonthlyDataPoint{
			Period: period,
			Label:  periodToLabel(period),
			Total:  total,
		})
		cur = cur.AddDate(0, 1, 0)
	}
	return result
}

func toJSONStringArray(points []MonthlyDataPoint) string {
	parts := make([]string, len(points))
	for i, p := range points {
		parts[i] = `"` + p.Label + `"`
	}
	return strings.Join(parts, ",")
}

func toJSONIntArray(points []MonthlyDataPoint) string {
	parts := make([]string, len(points))
	for i, p := range points {
		parts[i] = strconv.Itoa(p.Total)
	}
	return strings.Join(parts, ",")
}
