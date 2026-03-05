package pages

import (
	"context"
	"createmod/content"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/news"
	"createmod/internal/store"
	"html/template"
	"net/http"
	"strings"
	"time"

	"createmod/internal/server"
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

func NewsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := NewsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "News")
		d.Description = i18n.T(d.Language, "page.news.description")
		d.Slug = "/news"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

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
		d.HourlyViews, d.TotalViews24 = cachedHourlyStats(appStore, cacheService, "schematic_views")
		d.HourlyDL, d.TotalDL24 = cachedHourlyStats(appStore, cacheService, "schematic_downloads")

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
func cachedHourlyStats(appStore *store.Store, cacheService *cache.Service, table string) ([]HourlyStat, int) {
	key := "hourlyStats:" + table
	if v, ok := cacheService.Get(key); ok {
		if r, ok := v.(hourlyStatsResult); ok {
			return r.Stats, r.Total
		}
	}
	stats, total := loadHourlyStats(appStore, table)
	cacheService.SetWithTTL(key, hourlyStatsResult{Stats: stats, Total: total}, 5*time.Minute)
	return stats, total
}

// loadHourlyStats queries an aggregate table (schematic_views or
// schematic_downloads) and returns per-hour totals for the last 24 hours.
func loadHourlyStats(appStore *store.Store, table string) ([]HourlyStat, int) {
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	rows, err := appStore.Stats.HourlyStats(context.Background(), table, cutoff)

	// Build a map of hour -> count
	hourMap := make(map[string]int, 24)
	if err == nil {
		for _, r := range rows {
			hourMap[r.Hour] = int(r.Count)
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
