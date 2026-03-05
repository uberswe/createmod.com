package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"strings"

	"createmod/internal/server"
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
func GuidesNewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := GuidesNewData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "New Guide")
		d.Description = i18n.T(d.Language, "page.guides_create.description")
		d.Slug = "/guides/new"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(guidesNewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesCreateHandler handles POST /guides to insert a new guide record.
func GuidesCreateHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
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
			dest := "/guides/new?error=missing_title_or_content"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		authorID := authenticatedUserID(e)
		excerpt := ""
		if content != "" {
			excerpt = stripHTMLTags(content)
			if len(excerpt) > 180 {
				excerpt = excerpt[:180] + "..."
			}
		}

		newGuide := &store.Guide{
			Title:    title,
			Content:  content,
			Excerpt:  excerpt,
			VideoURL: video,
			WikiURL:  link,
			AuthorID: &authorID,
		}

		ctx := context.Background()
		if err := appStore.Guides.Create(ctx, newGuide); err != nil {
			slog.Warn("guides: failed to create", "error", err)
			return e.String(http.StatusInternalServerError, "failed to create guide")
		}

		// Award first_guide achievement asynchronously
		go awardFirstGuide(appStore, authorID)

		// Redirect to the newly created guide's detail page
		dest := "/guides/" + newGuide.ID
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
