package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	strip "github.com/grokify/html-strip-tags-go"
	"createmod/internal/server"
)

var collectionsTemplates = append([]string{
	"./template/collections.html",
}, commonTemplates...)

// CollectionItem is a lightweight view model for collections listing.
type CollectionItem struct {
	Title          string
	Description    string
	URL            string
	ImageURL       string
	SchematicCount int
	Views          int
	Featured       bool
	Published      bool
	IsOwner        bool
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
func CollectionsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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
		if tab == "mine" && !isAuthenticated(e) {
			tab = "public"
		}

		ctx := context.Background()
		items := make([]CollectionItem, 0, 64)

		if tab == "mine" {
			// Show all of the authenticated user's collections (published and private)
			colls, err := appStore.Collections.ListByAuthor(ctx, authenticatedUserID(e))
			if err == nil {
				for _, c := range colls {
					it := storeCollectionToItem(c, appStore)
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
				colls, err := appStore.Collections.ListPublished(ctx, 500, 0)
				if err == nil {
					type scored struct {
						item  CollectionItem
						score float64
					}
					scoredItems := make([]scored, 0, len(colls))
					for _, c := range colls {
						it := storeCollectionToItem(c, appStore)
						s := collectionTrendingScore(c.Created, float64(it.Views))
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

		return renderCollectionsPage(e, registry, cacheService, appStore, items, tab, q, page, pageSize)
	}
}

func storeCollectionToItem(c store.Collection, appStore *store.Store) CollectionItem {
	title := c.Title
	if title == "" {
		title = c.Name
	}
	link := "/collections/" + c.ID
	if c.Slug != "" {
		link = "/collections/" + c.Slug
	}
	imageURL := c.BannerURL
	if imageURL == "" {
		imageURL = c.CollageURL
	}
	var schematicCount int
	if appStore != nil {
		ids, err := appStore.Collections.GetSchematicIDs(context.Background(), c.ID)
		if err == nil {
			schematicCount = len(ids)
		}
	}
	return CollectionItem{
		Title:          title,
		Description:    strip.StripTags(c.Description),
		URL:            link,
		ImageURL:       imageURL,
		SchematicCount: schematicCount,
		Views:          c.Views,
		Featured:       c.Featured,
		Published:      c.Published,
	}
}

func renderCollectionsPage(e *server.RequestEvent, registry *server.Registry, cacheService *cache.Service, appStore *store.Store, items []CollectionItem, tab, q string, page, pageSize int) error {
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

	d.PopulateWithStore(e, appStore)
	d.Title = i18n.T(d.Language, "Collections")
	d.Description = i18n.T(d.Language, "Community-created collections of schematics")
	d.Slug = "/collections"
	d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

	html, err := registry.LoadFiles(collectionsTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
