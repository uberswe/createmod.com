package pages

import (
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/news"
	"createmod/internal/store"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
)

const newsTemplate = "./template/news.html"

var newsTemplates = append([]string{
	newsTemplate,
}, commonTemplates...)

// HourlyStat holds an hour label and a count for the stats charts.
type HourlyStat struct {
	Hour  string // e.g. "14:00"
	Count int
}

type NewsData struct {
	DefaultData
	Posts         []models.NewsPostListItem
	HourlyViews  []HourlyStat
	HourlyDL     []HourlyStat
	TotalViews24 int
	TotalDL24    int
}

func NewsHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := NewsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "News")
		d.Description = i18n.T(d.Language, "page.news.description")
		d.Slug = "/news"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		// Load news from embedded markdown files
		all, err := news.LoadAll(content.NewsFS, "news")
		if err == nil {
			posts := make([]models.NewsPostListItem, 0, len(all))
			for _, p := range all {
				posts = append(posts, models.NewsPostListItem{
					ID:             p.Slug,
					Title:          p.Title,
					Excerpt:        p.Excerpt,
					FirstParagraph: extractFirstParagraph(p.Body),
					URL:            p.URL,
					PostDate:       p.Date,
				})
			}
			d.Posts = posts
		}

		// Load 24-hour sitewide stats (cached for 5 minutes)
		d.HourlyViews, d.TotalViews24 = cachedHourlyStats(app, cacheService, "schematic_views")
		d.HourlyDL, d.TotalDL24 = cachedHourlyStats(app, cacheService, "schematic_downloads")

		html, err := registry.LoadFiles(newsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// extractFirstParagraph returns the content of the first <p>...</p> tag in
// rendered HTML. Returns empty if no paragraph is found.
func extractFirstParagraph(body template.HTML) template.HTML {
	s := string(body)
	start := strings.Index(s, "<p>")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start:], "</p>")
	if end < 0 {
		return ""
	}
	return template.HTML(s[start : start+end+4])
}

// hourlyStatsResult holds a cached snapshot of loadHourlyStats output.
type hourlyStatsResult struct {
	Stats []HourlyStat
	Total int
}

// cachedHourlyStats returns hourly stats from cache (5-min TTL) or queries the DB.
func cachedHourlyStats(app *pocketbase.PocketBase, cacheService *cache.Service, table string) ([]HourlyStat, int) {
	key := "hourlyStats:" + table
	if v, ok := cacheService.Get(key); ok {
		if r, ok := v.(hourlyStatsResult); ok {
			return r.Stats, r.Total
		}
	}
	stats, total := loadHourlyStats(app, table)
	cacheService.SetWithTTL(key, hourlyStatsResult{Stats: stats, Total: total}, 5*time.Minute)
	return stats, total
}

// loadHourlyStats queries an aggregate table (schematic_views or
// schematic_downloads) and returns per-hour totals for the last 24 hours.
func loadHourlyStats(app *pocketbase.PocketBase, table string) ([]HourlyStat, int) {
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")

	type row struct {
		H string  `db:"h" json:"h"`
		V float64 `db:"v" json:"v"`
	}

	var rows []row
	err := app.DB().NewQuery(fmt.Sprintf(
		"SELECT strftime('%%Y-%%m-%%d %%H', created) AS h, SUM(count) AS v FROM %s WHERE created > {:cutoff} AND type = 0 GROUP BY h ORDER BY h ASC",
		table,
	)).Bind(dbx.Params{"cutoff": cutoff}).All(&rows)

	// Build a map of hour -> count
	hourMap := make(map[string]int, 24)
	if err == nil {
		for _, r := range rows {
			hourMap[r.H] = int(r.V)
		}
	}

	// Generate 24 slots from (now-23h) to now
	stats := make([]HourlyStat, 0, 24)
	total := 0
	for i := 23; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Hour)
		key := t.Format("2006-01-02 15")
		label := t.Format("15:00")
		count := hourMap[key]
		total += count
		stats = append(stats, HourlyStat{Hour: label, Count: count})
	}
	return stats, total
}
