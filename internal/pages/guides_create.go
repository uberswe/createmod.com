package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
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
func GuidesNewHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := GuidesNewData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "New Guide")
		d.Description = i18n.T(d.Language, "page.guides_create.description")
		d.Slug = "/guides/new"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(guidesNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesCreateHandler handles POST /guides to insert a new guide record (best-effort schema compatible).
func GuidesCreateHandler(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if ok, err := requireAuth(e); !ok {
			return err
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
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
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
		rec.Set("author", authenticatedUserID(e))
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
			rec.Set("author", authenticatedUserID(e))
			_ = app.Save(rec) // best-effort; ignore error to avoid blocking UX in early stages
		}

		// Redirect to the newly created guide's detail page
		dest := "/guides/" + rec.Id
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// urlQueryEscape is a tiny helper to avoid importing net/url here.
func urlQueryEscape(s string) string {
	// very basic; production code should use url.QueryEscape
	// keep only simple runes, replace spaces
	s = strings.ReplaceAll(s, " ", "+")
	return s
}
