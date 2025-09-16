package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

var collectionsTemplates = append([]string{
	"./template/collections.html",
}, commonTemplates...)

// CollectionItem is a lightweight view model for collections listing.
// It is intentionally minimal until the PocketBase schema is finalized.
type CollectionItem struct {
	Title       string
	Description string
	URL         string
	Views       int
	Featured    bool
}

type CollectionsData struct {
	DefaultData
	Items    []CollectionItem
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
	Query    string
	Sort     string
}

// CollectionsHandler renders a basic listing of collections with pagination and optional search.
// If the underlying PocketBase collection is not available yet, it renders an empty list gracefully.
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
		// Sort parameter: featured | views | recent (default featured)
		s := strings.TrimSpace(strings.ToLower(e.Request.URL.Query().Get("s")))
		if s == "" {
			s = "featured"
		}
		if s != "featured" && s != "views" && s != "recent" {
			s = "featured"
		}

		// Try to query PB collection named "collections" (placeholder name).
		// If not found or error, continue with empty items.
		items := make([]CollectionItem, 0, 64)
		if coll, err := app.FindCollectionByNameOrId("collections"); err == nil && coll != nil {
			// Fetch recent entries; adapt filter later as schema stabilizes
			recs, err := app.FindRecordsByFilter(coll.Id, "deleted = null || deleted = ''", "-created", 500, 0)
			if err == nil {
				for _, r := range recs {
					title := r.GetString("title")
					if title == "" {
						title = r.GetString("name")
					}
					desc := r.GetString("description")
					// Build a URL; if schema has slug, prefer it, else use record id
					slug := r.GetString("slug")
					link := "/collections/" + r.Id
					if slug != "" {
						link = "/collections/" + slug
					}
					it := CollectionItem{Title: title, Description: desc, URL: link, Views: r.GetInt("views"), Featured: r.GetBool("featured")}
					if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
						continue
					}
					items = append(items, it)
				}
			}
		}

		// Apply sorting according to 's'
		switch s {
		case "views":
			sort.SliceStable(items, func(i, j int) bool {
				if items[i].Views != items[j].Views {
					return items[i].Views > items[j].Views
				}
				// secondary: featured first when equal views
				if items[i].Featured != items[j].Featured {
					return items[i].Featured && !items[j].Featured
				}
				return false
			})
		case "recent":
			// items are already in recent order from the PB query; do nothing
		default: // "featured"
			sort.SliceStable(items, func(i, j int) bool {
				if items[i].Featured != items[j].Featured {
					return items[i].Featured && !items[j].Featured
				}
				return false
			})
		}

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
			Page:     page,
			PageSize: pageSize,
			HasPrev:  hasPrev,
			HasNext:  hasNext,
			Query:    q,
			Sort:     s,
		}
		if d.HasPrev {
			d.PrevURL = "/collections?p=" + strconv.Itoa(page-1)
			if q != "" {
				d.PrevURL += "&q=" + url.QueryEscape(q)
			}
			if s != "" {
				d.PrevURL += "&s=" + url.QueryEscape(s)
			}
		}
		if d.HasNext {
			d.NextURL = "/collections?p=" + strconv.Itoa(page+1)
			if q != "" {
				d.NextURL += "&q=" + url.QueryEscape(q)
			}
			if s != "" {
				d.NextURL += "&s=" + url.QueryEscape(s)
			}
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
}
