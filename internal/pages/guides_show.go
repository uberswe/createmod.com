package pages

import (
	"createmod/internal/cache"
	"createmod/internal/translation"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"html/template"
	"net/http"
	"time"
)

var guidesShowTemplates = append([]string{
	"./template/guides_show.html",
}, commonTemplates...)

// GuideShowData holds data for the guide detail page.
type GuideShowData struct {
	DefaultData
	GuideTitle  string
	Content     template.HTML // rendered HTML content
	Excerpt     string
	VideoURL    string
	WikiURL     string
	Views       int
	AuthorName  string
	IsOwner      bool
	GuideID      string
	NotFound     bool
	IsTranslated bool
}

// GuidesShowHandler renders an individual guide page by record ID.
func GuidesShowHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service, translationService *translation.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		d := GuideShowData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)
		d.Slug = "/guides/" + id

		coll, err := app.FindCollectionByNameOrId("guides")
		if err != nil || coll == nil {
			d.NotFound = true
			d.Title = "Guide not found"
			d.Description = "We couldn't find this guide."
			html, err := registry.LoadFiles(guidesShowTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			// Try finding by title slug (fallback)
			recs, ferr := app.FindRecordsByFilter(coll.Id, "title = {:t}", "-created", 1, 0, dbx.Params{"t": id})
			if ferr == nil && len(recs) > 0 {
				rec = recs[0]
			}
		}

		if rec == nil {
			d.NotFound = true
			d.Title = "Guide not found"
			d.Description = "We couldn't find this guide."
			html, err := registry.LoadFiles(guidesShowTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d.GuideID = rec.Id
		d.GuideTitle = rec.GetString("title")
		if d.GuideTitle == "" {
			d.GuideTitle = rec.GetString("name")
		}
		d.Excerpt = rec.GetString("excerpt")
		d.VideoURL = rec.GetString("video_url")
		d.WikiURL = rec.GetString("wiki_url")

		// Try multiple content fields
		content := rec.GetString("content")
		if content == "" {
			content = rec.GetString("content_markdown")
		}
		if content == "" {
			content = rec.GetString("markdown")
		}
		d.Content = template.HTML(content)

		// Owner check
		if e.Auth != nil && rec.GetString("author") == e.Auth.Id {
			d.IsOwner = true
		}

		// Load author name
		if authorID := rec.GetString("author"); authorID != "" {
			if u := findUserFromID(app, authorID); u != nil {
				d.AuthorName = u.Username
			}
		}

		// Increment views with IP-based deduplication (1-hour window)
		currentViews := rec.GetInt("views")
		clientIP := e.RealIP()
		ipKey := fmt.Sprintf("viewip:%s:guide:%s", clientIP, rec.Id)
		if clientIP != "" && cacheService != nil {
			if _, already := cacheService.Get(ipKey); already {
				// Same IP viewed this guide recently — skip increment
				d.Views = currentViews
			} else {
				cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
				rec.Set("views", currentViews+1)
				if err := app.Save(rec); err == nil {
					d.Views = currentViews + 1
				} else {
					d.Views = currentViews
				}
			}
		} else {
			d.Views = currentViews
		}

		// Translation: show translated content if user's language is not English
		showOriginal := e.Request.URL.Query().Get("lang") == "original"
		if !showOriginal && translationService != nil && d.Language != "" && d.Language != "en" {
			t := translationService.GetGuideTranslation(app, cacheService, rec.Id, d.Language)
			if t != nil && t.Title != "" {
				d.GuideTitle = t.Title
				if t.Description != "" {
					d.Excerpt = t.Description
				}
				if t.Content != "" {
					d.Content = template.HTML(t.Content)
				}
				d.IsTranslated = true
			}
		}

		// SEO
		d.Title = d.GuideTitle
		if d.Title == "" {
			d.Title = "Guide"
		}
		if d.Excerpt != "" {
			d.Description = d.Excerpt
		} else {
			d.Description = "Guide on CreateMod.com"
		}

		html, err := registry.LoadFiles(guidesShowTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
