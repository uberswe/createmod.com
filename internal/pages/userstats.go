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
	Period string // "YYYYMM"
	Label  string // "Jan 2025"
	Total  int
}

type UserStatsData struct {
	DefaultData
	MonthlyViews       []MonthlyDataPoint
	MonthlyDownloads   []MonthlyDataPoint
	ViewLabelsJSON     html.JS
	ViewValuesJSON     html.JS
	DownloadLabelsJSON html.JS
	DownloadValuesJSON html.JS
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

		now := time.Now().UTC()
		start := now.AddDate(0, -11, 0)

		monthlyStats, err := appStore.Stats.MonthlyUserStats(context.Background(), userID, 12)
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

		html, err := registry.LoadFiles(userStatsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
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
