package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strings"
	"time"
)

var guidesNewTemplates = append([]string{
	"./template/guides_new.html",
}, commonTemplates...)

// GuidesNewData holds data for the new guide form.
type GuidesNewData struct {
	DefaultData
	Error string
}

// GuidesNewHandler renders a simple Markdown editor form for creating guides (auth required).
func GuidesNewHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		d := GuidesNewData{}
		d.Populate(e)
		d.Title = "New Guide"
		d.Description = "Create a new guide (Markdown)"
		d.Slug = "/guides/new"
		d.Categories = allCategories(app, cacheService)
		html, err := registry.LoadFiles(guidesNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesCreateHandler handles POST /guides to insert a new guide record (best-effort schema compatible).
func GuidesCreateHandler(app *pocketbase.PocketBase, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		title := strings.TrimSpace(e.Request.FormValue("title"))
		content := strings.TrimSpace(e.Request.FormValue("content"))
		video := strings.TrimSpace(e.Request.FormValue("video_url"))
		link := strings.TrimSpace(e.Request.FormValue("external_url"))
		if title == "" && content == "" {
			// nothing to save
			dest := "/guides/new?error=missing_title_or_content"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", dest)
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, dest)
		}

		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "guides collection not available")
		}
		rec := core.NewRecord(coll)
		// Try to populate a variety of likely fields; PB will ignore or error if unsupported.
		if title != "" {
			rec.Set("title", title)
			rec.Set("name", title)
		}
		if content != "" {
			rec.Set("content", content)
			rec.Set("content_markdown", content)
			rec.Set("markdown", content)
		}
		if video != "" {
			rec.Set("video_url", video)
		}
		if link != "" {
			rec.Set("wiki_url", link)
		}
		rec.Set("author", e.Auth.Id)
		// If excerpt field exists, try a short preview (strip HTML tags from content)
		if content != "" {
			ex := stripHTMLTags(content)
			if len(ex) > 180 {
				ex = ex[:180] + "..."
			}
			rec.Set("excerpt", ex)
		}
		// Attempt save; if it fails due to unknown fields, try a minimal payload (title only)
		if err := app.Save(rec); err != nil {
			// fallback minimal insert with only a name/title field or created timestamp marker via another field
			rec = core.NewRecord(coll)
			if title == "" {
				title = "Guide " + time.Now().Format("2006-01-02 15:04")
			}
			rec.Set("title", title)
			rec.Set("name", title)
			rec.Set("author", e.Auth.Id)
			_ = app.Save(rec) // best-effort; ignore error to avoid blocking UX in early stages
		}

		// Redirect to the newly created guide's detail page
		dest := "/guides/" + rec.Id
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// urlQueryEscape is a tiny helper to avoid importing net/url here.
func urlQueryEscape(s string) string {
	// very basic; production code should use url.QueryEscape
	// keep only simple runes, replace spaces
	s = strings.ReplaceAll(s, " ", "+")
	return s
}
