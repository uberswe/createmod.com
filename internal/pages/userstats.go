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

		// Monthly stats (existing)
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

		d.MonthlyViews = fillMissingMonths(views, start, now)
		d.MonthlyDownloads = fillMissingMonths(downloads, start, now)
		d.ViewLabelsJSON = html.JS(toJSONStringArray(d.MonthlyViews))
		d.ViewValuesJSON = html.JS(toJSONIntArray(d.MonthlyViews))
		d.DownloadLabelsJSON = html.JS(toJSONStringArray(d.MonthlyDownloads))
		d.DownloadValuesJSON = html.JS(toJSONIntArray(d.MonthlyDownloads))

		// Hourly aggregate stats (30 days)
		since := now.AddDate(0, 0, -30)

		cacheKey := fmt.Sprintf("user_analytics:%s", userID)
		type cachedHourly struct {
			Views, Downloads, VideoPlays, YTClicks, TimeOnPage, LayerViews string
			TotalViews, TotalDownloads                                     int
		}

		var hourly cachedHourly
		if cached, found := cacheService.Get(cacheKey); found {
			if ch, ok := cached.(cachedHourly); ok {
				hourly = ch
			}
		}

		if hourly.Views == "" {
			hViews, _ := appStore.Stats.HourlyUserViews(ctx, userID, since)
			hDownloads, _ := appStore.Stats.HourlyUserDownloads(ctx, userID, since)
			hVideoPlays, _ := appStore.Stats.HourlyUserEvents(ctx, userID, store.EventVideoPlay, since)
			hYTClicks, _ := appStore.Stats.HourlyUserEvents(ctx, userID, store.EventYouTubeClick, since)
			hTimeOnPage, _ := appStore.Stats.HourlyUserEvents(ctx, userID, store.EventTimeOnPage, since)
			hLayerViews, _ := appStore.Stats.HourlyUserEvents(ctx, userID, store.EventLayerViewer, since)

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
			cacheService.SetWithTTL(cacheKey, hourly, 15*time.Minute)
		}

		d.HourlyViewsJSON = html.JS(hourly.Views)
		d.HourlyDownloadsJSON = html.JS(hourly.Downloads)
		d.HourlyVideoPlaysJSON = html.JS(hourly.VideoPlays)
		d.HourlyYTClicksJSON = html.JS(hourly.YTClicks)
		d.HourlyTimeOnPageJSON = html.JS(hourly.TimeOnPage)
		d.HourlyLayerViewsJSON = html.JS(hourly.LayerViews)
		d.TotalViews30d = hourly.TotalViews
		d.TotalDownloads30d = hourly.TotalDownloads

		// VD ratio
		var siteAvg float64
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio"); ok {
			siteAvg = cached
		} else {
			siteAvg, _ = appStore.Stats.GetSiteAvgVDRatio(ctx)
			cacheService.SetFloat("site_avg_vd_ratio", siteAvg)
		}
		d.SiteAvgVDRatio = siteAvg

		var vdRatio float64
		if d.TotalViews30d > 0 {
			vdRatio = float64(d.TotalDownloads30d) / float64(d.TotalViews30d)
		}
		d.VDRatioPercent = fmt.Sprintf("%.2f%%", vdRatio*100)
		d.VDRatioBetter = vdRatio >= siteAvg

		// Schematic list with pagination
		d.PageSize = 20
		pageStr := e.Request.URL.Query().Get("page")
		d.Page, _ = strconv.Atoi(pageStr)
		if d.Page < 1 {
			d.Page = 1
		}
		offset := (d.Page - 1) * d.PageSize

		totalSchematics, _ := appStore.Stats.CountUserSchematics(ctx, userID)
		d.TotalSchematics = totalSchematics
		d.TotalPages = (totalSchematics + d.PageSize - 1) / d.PageSize
		d.HasPrevPage = d.Page > 1
		d.HasNextPage = d.Page < d.TotalPages
		d.PrevPage = d.Page - 1
		d.NextPage = d.Page + 1

		schematicStats, _ := appStore.Stats.ListSchematicStats(ctx, userID, d.PageSize, offset)
		cutoff := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
		for _, s := range schematicStats {
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
