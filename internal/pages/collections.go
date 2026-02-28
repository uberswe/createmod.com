package pages

import (
	"createmod/internal/cache"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
)

var collectionsTemplates = append([]string{
	"./template/collections.html",
}, commonTemplates...)

// CollectionItem is a lightweight view model for collections listing.
type CollectionItem struct {
	Title       string
	Description string
	URL         string
	Views       int
	Featured    bool
	Published   bool
	IsOwner     bool
}

type CollectionsData struct {
	DefaultData
	Items    []CollectionItem
	Tab      string
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
	Query    string
	Sort     string
}

// collectionTrendingScore computes a Reddit-style trending score for collections.
// Uses views as the engagement signal with a 12-month timescale.
func collectionTrendingScore(created time.Time, views float64) float64 {
	const timescale = 365 * 24 * 3600.0 // 12 months
	engagement := views
	order := math.Log10(math.Max(engagement, 1))
	seconds := created.Sub(trendingEpoch).Seconds()
	return order + seconds/timescale
}

// CollectionsHandler renders collections with two tabs: public (trending) and mine (user's own).
func CollectionsHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Pagination params
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		// Optional search query
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		qLower := strings.ToLower(q)

		// Tab: "public" (default) or "mine"
		tab := strings.TrimSpace(strings.ToLower(e.Request.URL.Query().Get("tab")))
		if tab != "mine" {
			tab = "public"
		}
		// "mine" requires authentication; fall back to "public"
		if tab == "mine" && e.Auth == nil {
			tab = "public"
		}

		items := make([]CollectionItem, 0, 64)

		coll, collErr := app.FindCollectionByNameOrId("collections")
		if collErr != nil || coll == nil {
			// No PB schema available; render empty list
			return renderCollectionsPage(e, app, registry, cacheService, items, tab, q, page, pageSize)
		}

		if tab == "mine" {
			// Show all of the authenticated user's collections (published and private)
			recs, err := app.FindRecordsByFilter(coll.Id, "deleted = '' && author = {:author}", "-created", 500, 0, map[string]any{"author": e.Auth.Id})
			if err == nil {
				for _, r := range recs {
					it := recordToCollectionItem(r)
					it.IsOwner = true
					if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
						continue
					}
					items = append(items, it)
				}
			}
		} else {
			// Public tab: show published collections sorted by trending score.
			// Try cache first.
			const cacheKey = "collections:public:trending"
			if cached, ok := cacheService.Get(cacheKey); ok {
				if cachedItems, ok := cached.([]CollectionItem); ok {
					// Filter by search query if provided
					for _, it := range cachedItems {
						if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
							continue
						}
						items = append(items, it)
					}
				}
			}

			if len(items) == 0 {
				recs, err := app.FindRecordsByFilter(coll.Id, "deleted = '' && published = true", "-created", 500, 0)
				if err == nil {
					type scored struct {
						item  CollectionItem
						score float64
					}
					scoredItems := make([]scored, 0, len(recs))
					for _, r := range recs {
						it := recordToCollectionItem(r)
						s := collectionTrendingScore(r.GetDateTime("created").Time(), float64(it.Views))
						scoredItems = append(scoredItems, scored{item: it, score: s})
					}
					sort.SliceStable(scoredItems, func(i, j int) bool {
						return scoredItems[i].score > scoredItems[j].score
					})
					allItems := make([]CollectionItem, 0, len(scoredItems))
					for _, si := range scoredItems {
						allItems = append(allItems, si.item)
					}
					// Cache the full sorted list for 30 minutes
					cacheService.SetWithTTL(cacheKey, allItems, 30*time.Minute)

					// Apply search filter
					for _, it := range allItems {
						if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
							continue
						}
						items = append(items, it)
					}
				}
			}
		}

		return renderCollectionsPage(e, app, registry, cacheService, items, tab, q, page, pageSize)
	}
}

func recordToCollectionItem(r *core.Record) CollectionItem {
	title := r.GetString("title")
	if title == "" {
		title = r.GetString("name")
	}
	slug := r.GetString("slug")
	link := "/collections/" + r.Id
	if slug != "" {
		link = "/collections/" + slug
	}
	return CollectionItem{
		Title:       title,
		Description: r.GetString("description"),
		URL:         link,
		Views:       r.GetInt("views"),
		Featured:    r.GetBool("featured"),
		Published:   r.GetBool("published"),
	}
}

func renderCollectionsPage(e *core.RequestEvent, app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, items []CollectionItem, tab, q string, page, pageSize int) error {
	// Pagination on items
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	paged := items[start:end]
	hasPrev := page > 1
	hasNext := len(items) > end

	d := CollectionsData{
		Items:    paged,
		Tab:      tab,
		Page:     page,
		PageSize: pageSize,
		HasPrev:  hasPrev,
		HasNext:  hasNext,
		Query:    q,
	}

	buildURL := func(p int) string {
		u := "/collections?tab=" + url.QueryEscape(tab) + "&p=" + strconv.Itoa(p)
		if q != "" {
			u += "&q=" + url.QueryEscape(q)
		}
		return u
	}
	if d.HasPrev {
		d.PrevURL = buildURL(page - 1)
	}
	if d.HasNext {
		d.NextURL = buildURL(page + 1)
	}

	d.Populate(e)
	d.Title = "Collections"
	d.Description = "Community-created collections of schematics"
	d.Slug = "/collections"
	d.Categories = allCategories(app, cacheService)

	html, err := registry.LoadFiles(collectionsTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
