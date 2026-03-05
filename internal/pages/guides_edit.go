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

var guidesEditTemplates = append([]string{
	"./template/guides_edit.html",
}, commonTemplates...)

// GuideEditData holds data for the guide edit form.
type GuideEditData struct {
	DefaultData
	GuideID    string
	GuideTitle string
	Content    string
	VideoURL   string
	WikiURL    string
	Excerpt    string
	Error      string
}

// GuidesEditHandler renders the edit form for an existing guide (owner-only).
func GuidesEditHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides/"+id))
		}

		d := GuideEditData{
			GuideID:    guide.ID,
			GuideTitle: guide.Title,
			Content:    guide.Content,
			VideoURL:   guide.VideoURL,
			WikiURL:    guide.WikiURL,
			Excerpt:    guide.Excerpt,
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Edit Guide")
		d.Description = i18n.T(d.Language, "page.guides_edit.description")
		d.Slug = "/guides/" + id + "/edit"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(guidesEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesUpdateHandler handles POST /guides/{id} to update an existing guide (owner-only).
func GuidesUpdateHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides/"+id))
		}

		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}

		title := strings.TrimSpace(e.Request.FormValue("title"))
		content := strings.TrimSpace(e.Request.FormValue("content"))
		video := strings.TrimSpace(e.Request.FormValue("video_url"))
		link := strings.TrimSpace(e.Request.FormValue("external_url"))
		excerpt := strings.TrimSpace(e.Request.FormValue("excerpt"))

		if title != "" {
			guide.Title = title
		}
		guide.Content = content
		if video != "" {
			guide.VideoURL = video
		}
		if link != "" {
			guide.WikiURL = link
		}
		if excerpt != "" {
			guide.Excerpt = excerpt
		} else if content != "" {
			ex := stripHTMLTags(content)
			if len(ex) > 180 {
				ex = ex[:180] + "..."
			}
			guide.Excerpt = ex
		}

		if err := appStore.Guides.Update(ctx, guide); err != nil {
			slog.Warn("guides: failed to update", "error", err)
		}

		dest := "/guides/" + id
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// GuidesDeleteHandler handles POST /guides/{id}/delete to delete a guide (owner-only).
func GuidesDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		ctx := context.Background()

		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}
		if guide.AuthorID == nil || *guide.AuthorID != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		if err := appStore.Guides.Delete(ctx, guide.ID); err != nil {
			slog.Warn("guides: failed to delete", "error", err)
		}

		dest := "/guides"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}

// stripHTMLTags removes HTML tags from a string for generating plain-text excerpts.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}
