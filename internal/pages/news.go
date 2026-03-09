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

// DailyStat holds a day label and a count for the stats charts.
type DailyStat struct {
	Day   string // e.g. "Mon", "Tue"
	Date  string // e.g. "Mar 03"
	Count int
}

type NewsData struct {
	DefaultData
	Posts        []models.NewsPostListItem
	DailyViews  []DailyStat
	DailyDL     []DailyStat
	TotalViews7 int
	TotalDL7    int
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

		// Load 7-day sitewide stats (cached for 5 minutes)
		d.DailyViews, d.TotalViews7 = cachedDailyStats(appStore, cacheService, "schematic_views")
		d.DailyDL, d.TotalDL7 = cachedDailyStats(appStore, cacheService, "schematic_downloads")

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

// dailyStatsResult holds a cached snapshot of loadDailyStats output.
type dailyStatsResult struct {
	Stats []DailyStat
	Total int
}

// cachedDailyStats returns daily stats from cache (5-min TTL) or queries the DB.
func cachedDailyStats(appStore *store.Store, cacheService *cache.Service, table string) ([]DailyStat, int) {
	key := "dailyStats:" + table
	if v, ok := cacheService.Get(key); ok {
		if r, ok := v.(dailyStatsResult); ok {
			return r.Stats, r.Total
		}
	}
	stats, total := loadDailyStats(appStore, table)
	cacheService.SetWithTTL(key, dailyStatsResult{Stats: stats, Total: total}, 5*time.Minute)
	return stats, total
}

// loadDailyStats queries an aggregate table (schematic_views or
// schematic_downloads) and returns per-day totals for the last 7 days.
func loadDailyStats(appStore *store.Store, table string) ([]DailyStat, int) {
	now := time.Now().UTC()
	cutoff := now.Add(-7 * 24 * time.Hour)

	rows, err := appStore.Stats.HourlyStats(context.Background(), table, cutoff)

	// Aggregate hourly rows into daily buckets
	dayMap := make(map[string]int, 7)
	if err == nil {
		for _, r := range rows {
			// r.Hour is "YYYY-MM-DD HH", take the date part
			if len(r.Hour) >= 10 {
				day := r.Hour[:10]
				dayMap[day] += int(r.Count)
			}
		}
	}

	// Generate 7 slots from (now-6d) to today
	stats := make([]DailyStat, 0, 7)
	total := 0
	for i := 6; i >= 0; i-- {
		t := now.AddDate(0, 0, -i)
		key := t.Format("2006-01-02")
		count := dayMap[key]
		total += count
		stats = append(stats, DailyStat{
			Day:   t.Format("Mon"),
			Date:  t.Format("Jan 02"),
			Count: count,
		})
	}
	return stats, total
}
