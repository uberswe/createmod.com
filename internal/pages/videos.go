package pages

import (
	"createmod/internal/cache"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var videosTemplates = append([]string{
	"./template/videos.html",
}, commonTemplates...)

// VideoItem represents a unique YouTube video referenced by a schematic.
type VideoItem struct {
	ID           string // YouTube ID
	Title        string
	ThumbnailURL string
	VideoURL     string
	SchematicURL string
}

type VideosData struct {
	DefaultData
	Videos   []VideoItem
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
	Query    string
}

var (
	// regexes to catch common YouTube URL forms
	reWatch  = regexp.MustCompile(`(?:v=)([A-Za-z0-9_-]{6,})`)
	reShort  = regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]{6,})`)
	reShorts = regexp.MustCompile(`youtube\.com/shorts/([A-Za-z0-9_-]{6,})`)
)

func youtubeID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Parse to normalize query when possible
	if u, err := url.Parse(s); err == nil && u != nil {
		// try query v
		if v := u.Query().Get("v"); v != "" {
			return v
		}
	}
	if m := reWatch.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if m := reShort.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if m := reShorts.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	return ""
}

func youtubeThumb(id string) string {
	if id == "" {
		return ""
	}
	// using hqdefault for decent quality
	return "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg"
}

// VideosHandler renders a page of unique YouTube videos referenced by schematics.
func VideosHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		coll, err := app.FindCollectionByNameOrId("schematics")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "schematics collection not available")
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

		// Fetch recently created schematics with a non-empty video field, only moderated & published
		recs, err := app.FindRecordsByFilter(coll.Id, "deleted = '' && moderated = true && video != '' && (scheduled_at = null || scheduled_at <= {:now})", "-created", 500, 0, dbx.Params{"now": time.Now()})
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to query schematics with videos")
		}
		// Deduplicate by YouTube ID while preserving order
		seen := make(map[string]bool)
		items := make([]VideoItem, 0, len(recs))
		for _, r := range recs {
			vid := youtubeID(r.GetString("video"))
			if vid == "" || seen[vid] {
				continue
			}
			seen[vid] = true
			name := r.GetString("name")
			title := r.GetString("title")
			it := VideoItem{
				ID:           vid,
				Title:        title,
				ThumbnailURL: youtubeThumb(vid),
				VideoURL:     "https://www.youtube.com/watch?v=" + vid,
				SchematicURL: "/schematics/" + name,
			}
			// Filter by q if provided (title contains q, case-insensitive)
			if q != "" && !strings.Contains(strings.ToLower(it.Title), qLower) {
				continue
			}
			items = append(items, it)
		}

		// Apply pagination on unique videos
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

		d := VideosData{
			Videos:   paged,
			Page:     page,
			PageSize: pageSize,
			HasPrev:  hasPrev,
			HasNext:  hasNext,
			Query:    q,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/videos?p=%d", page-1)
			if q != "" {
				d.PrevURL += "&q=" + url.QueryEscape(q)
			}
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/videos?p=%d", page+1)
			if q != "" {
				d.NextURL += "&q=" + url.QueryEscape(q)
			}
		}

		d.Populate(e)
		d.Title = "Videos"
		d.Description = "Unique YouTube videos referenced by schematics"
		d.Slug = "/videos"
		d.Categories = allCategories(app, cacheService)

		html, err := registry.LoadFiles(videosTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
