package pages

import (
	"createmod/internal/cache"
	"fmt"
	html "html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
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

func UserStatsHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Require auth
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}

		d := UserStatsData{}
		d.Populate(e)
		d.Title = "Statistics"
		d.Description = "Your schematic statistics."
		d.Slug = "/settings/statistics"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)

		now := time.Now().UTC()
		start := now.AddDate(0, -11, 0)
		startPeriod := fmt.Sprintf("%d%02d", start.Year(), int(start.Month()))

		views := fetchMonthlyStats(app, "schematic_views", e.Auth.Id, startPeriod)
		downloads := fetchMonthlyStats(app, "schematic_downloads", e.Auth.Id, startPeriod)

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

type monthlyRow struct {
	Period string
	Total  int
}

func fetchMonthlyStats(app *pocketbase.PocketBase, collectionName string, userID string, startPeriod string) []MonthlyDataPoint {
	var rows []monthlyRow
	err := app.RecordQuery(collectionName).
		Select(collectionName+".period as period", "SUM("+collectionName+".count) as total").
		From(collectionName).
		LeftJoin("schematics", dbx.NewExp(collectionName+".schematic = schematics.id")).
		Where(dbx.NewExp(
			collectionName+".type = 2 AND schematics.author = {:userId} AND "+collectionName+".period >= {:startPeriod}",
			dbx.Params{"userId": userID, "startPeriod": startPeriod},
		)).
		GroupBy(collectionName + ".period").
		OrderBy(collectionName + ".period ASC").
		All(&rows)
	if err != nil {
		return nil
	}

	points := make([]MonthlyDataPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, MonthlyDataPoint{
			Period: r.Period,
			Label:  periodToLabel(r.Period),
			Total:  r.Total,
		})
	}
	return points
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
