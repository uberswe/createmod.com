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
func GuidesEditHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "guides collection not available")
		}
		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if rec.GetString("author") != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides/"+id))
		}

		content := rec.GetString("content")
		if content == "" {
			content = rec.GetString("content_markdown")
		}
		if content == "" {
			content = rec.GetString("markdown")
		}

		d := GuideEditData{
			GuideID:    rec.Id,
			GuideTitle: rec.GetString("title"),
			Content:    content,
			VideoURL:   rec.GetString("video_url"),
			WikiURL:    rec.GetString("wiki_url"),
			Excerpt:    rec.GetString("excerpt"),
		}
		if d.GuideTitle == "" {
			d.GuideTitle = rec.GetString("name")
		}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Edit Guide")
		d.Description = i18n.T(d.Language, "page.guides_edit.description")
		d.Slug = "/guides/" + id + "/edit"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)

		html, err := registry.LoadFiles(guidesEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// GuidesUpdateHandler handles POST /guides/{id} to update an existing guide (owner-only).
func GuidesUpdateHandler(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "guides collection not available")
		}
		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		// Owner check
		if rec.GetString("author") != authenticatedUserID(e) {
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
			rec.Set("title", title)
			rec.Set("name", title)
		}
		rec.Set("content", content)
		rec.Set("content_markdown", content)
		rec.Set("markdown", content)
		if video != "" {
			rec.Set("video_url", video)
		}
		if link != "" {
			rec.Set("wiki_url", link)
		}
		if excerpt != "" {
			rec.Set("excerpt", excerpt)
		} else if content != "" {
			ex := content
			// Strip HTML tags for auto-excerpt
			ex = stripHTMLTags(ex)
			if len(ex) > 180 {
				ex = ex[:180] + "..."
			}
			rec.Set("excerpt", ex)
		}

		if err := app.Save(rec); err != nil {
			app.Logger().Warn("guides: failed to update", "error", err)
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
func GuidesDeleteHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		id := e.Request.PathValue("id")
		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "guides collection not available")
		}
		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}
		if rec.GetString("author") != authenticatedUserID(e) {
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/guides"))
		}

		if err := app.Delete(rec); err != nil {
			app.Logger().Warn("guides: failed to delete", "error", err)
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
