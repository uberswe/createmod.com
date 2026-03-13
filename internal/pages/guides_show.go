package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"createmod/internal/server"
	"github.com/sym01/htmlsanitizer"
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
	GuideTitle   string
	Content      template.HTML // rendered HTML content
	Excerpt      string
	VideoURL     string
	WikiURL      string
	BannerURL    string
	Views        int
	AuthorName   string
	IsOwner      bool
	GuideID      string
	NotFound     bool
	IsTranslated bool
}

// GuidesShowHandler renders an individual guide page by record ID.
func GuidesShowHandler(registry *server.Registry, cacheService *cache.Service, translationService *translation.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		id := e.Request.PathValue("id")
		ctx := context.Background()

		d := GuideShowData{}
		d.PopulateWithStore(e, appStore)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/guides/" + id
		// Breadcrumbs set after guide is loaded below

		// Try finding by ID first, then by slug
		guide, err := appStore.Guides.GetByID(ctx, id)
		if err != nil || guide == nil {
			guide, err = appStore.Guides.GetBySlug(ctx, id)
		}

		if guide == nil {
			d.NotFound = true
			d.Title = i18n.T(d.Language, "Guide not found")
			d.Description = i18n.T(d.Language, "We couldn't find this guide.")
			html, err := registry.LoadFiles(guidesShowTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusNotFound, html)
		}

		d.GuideID = guide.ID
		d.GuideTitle = guide.Title
		d.Excerpt = guide.Excerpt
		if d.Excerpt == "" {
			d.Excerpt = guide.Description
		}
		d.VideoURL = guide.VideoURL
		d.WikiURL = guide.WikiURL
		d.BannerURL = guide.BannerURL
		sanitizer := htmlsanitizer.NewHTMLSanitizer()
		sanitizedContent, sanitizeErr := sanitizer.SanitizeString(guide.Content)
		if sanitizeErr != nil {
			sanitizedContent = template.HTMLEscapeString(guide.Content)
		}
		d.Content = template.HTML(sanitizedContent)

		// Owner check
		if isAuthenticated(e) && guide.AuthorID != nil && *guide.AuthorID == authenticatedUserID(e) {
			d.IsOwner = true
		}

		// Load author name
		if guide.AuthorID != nil && *guide.AuthorID != "" {
			if u := findUserFromStore(appStore, *guide.AuthorID); u != nil {
				d.AuthorName = u.Username
			}
		}

		// Increment views with IP-based deduplication (1-hour window)
		currentViews := guide.Views
		clientIP := e.RealIP()
		ipKey := fmt.Sprintf("viewip:%s:guide:%s", clientIP, guide.ID)
		if clientIP != "" && cacheService != nil {
			if _, already := cacheService.Get(ipKey); already {
				d.Views = currentViews
			} else {
				cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
				if err := appStore.Guides.IncrementViews(ctx, guide.ID); err == nil {
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
			t := translationService.GetGuideTranslationCached(cacheService, guide.ID, d.Language)
			if t != nil && t.Title != "" {
				d.GuideTitle = t.Title
				if t.Description != "" {
					d.Excerpt = t.Description
				}
				if t.Content != "" {
					sanitizedTranslation, sanitizeErr := sanitizer.SanitizeString(t.Content)
					if sanitizeErr != nil {
						sanitizedTranslation = template.HTMLEscapeString(t.Content)
					}
					d.Content = template.HTML(sanitizedTranslation)
				}
				d.IsTranslated = true
			}
		}

		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Guides"), "/guides", d.GuideTitle)

		// SEO
		d.Title = d.GuideTitle
		if d.Title == "" {
			d.Title = i18n.T(d.Language, "page.guide.title")
		}
		if d.Excerpt != "" {
			d.Description = d.Excerpt
		} else {
			d.Description = i18n.T(d.Language, "page.guides_show.description")
		}

		html, err := registry.LoadFiles(guidesShowTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
