package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/outurl"
	"createmod/internal/store"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var guidesTemplates = append([]string{
	"./template/guides.html",
}, commonTemplates...)

// GuideItem represents a lightweight guide entry.
type GuideItem struct {
	ID       string
	Title    string
	Excerpt  string
	URL      string // internal detail page URL
	VideoURL string // optional external video link wrapped via /out?url=...&guide={id}
	Views    int
}

type GuidesData struct {
	DefaultData
	Items    []GuideItem
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
	Query    string
}

// GuidesHandler renders a simple listing of guides with pagination and optional search by title.
func GuidesHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, outSecret string, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "guides collection not available")
		}
		// Pagination params
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		// Query filter
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		qLower := strings.ToLower(q)

		// Fetch a recent batch; keep server logic simple and filter/paginate in-memory (like Videos page)
		recs, err := app.FindRecordsByFilter(coll.Id, "1=1", "-created", 500, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to query guides")
		}

		items := make([]GuideItem, 0, len(recs))
		for _, r := range recs {
			title := r.GetString("title")
			if title == "" {
				title = r.GetString("name")
			}
			excerpt := r.GetString("excerpt")
			// Link to the internal detail page
			link := "/guides/" + r.Id
			// Optional video url
			video := r.GetString("video_url")
			videoWrapped := ""
			if strings.HasPrefix(strings.ToLower(video), "http://") || strings.HasPrefix(strings.ToLower(video), "https://") {
				videoWrapped = outurl.BuildPathWithSource(video, outSecret, "guide", r.Id)
			}
			it := GuideItem{ID: r.Id, Title: title, Excerpt: excerpt, URL: link, VideoURL: videoWrapped, Views: r.GetInt("views")}
			if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
				continue
			}
			items = append(items, it)
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

		d := GuidesData{
			Items:    paged,
			Page:     page,
			PageSize: pageSize,
			HasPrev:  hasPrev,
			HasNext:  hasNext,
			Query:    q,
		}
		if d.HasPrev {
			d.PrevURL = "/guides?p=" + strconv.Itoa(page-1)
			if q != "" {
				d.PrevURL += "&q=" + url.QueryEscape(q)
			}
		}
		if d.HasNext {
			d.NextURL = "/guides?p=" + strconv.Itoa(page+1)
			if q != "" {
				d.NextURL += "&q=" + url.QueryEscape(q)
			}
		}

		d.Populate(e)
		d.Title = i18n.T(d.Language, "Guides")
		d.Description = i18n.T(d.Language, "Guides for the Create mod and Minecraft")
		d.Slug = "/guides"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(guidesTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
