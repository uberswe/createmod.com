package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"createmod/internal/server"
	"github.com/sym01/htmlsanitizer"
	"html/template"
	"net/http"
	"time"
)

var collectionsShowTemplates = append([]string{
	"./template/collections_show.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

// CollectionsShowData represents data for a single collection view page.
type CollectionsShowData struct {
	DefaultData
	TitleText       string
	DescriptionText string // raw description from DB (may be empty)
	DescriptionHTML template.HTML
	BannerURL       string
	Views           int
	Featured        bool
	Published       bool
	IsOwner         bool
	Schematics      []models.Schematic
	ShareURL        string
	CollectionID    string
	AuthorName      string
	IsTranslated    bool
	ModInfoList     []ModInfo
	SchematicCount  int
}

// CollectionsShowHandler renders a basic collection detail page by slug or id.
func CollectionsShowHandler(registry *server.Registry, cacheService *cache.Service, translationService *translation.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		slug := e.Request.PathValue("slug")

		ctx := context.Background()

		d := CollectionsShowData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Slug = "/collections/" + slug

		// Try to find by slug first, fallback to id
		coll, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || coll == nil {
			coll, err = appStore.Collections.GetByID(ctx, slug)
		}

		if coll != nil {
			d.Published = coll.Published
			d.CollectionID = coll.ID
			d.TitleText = coll.Title
			if d.TitleText == "" {
				d.TitleText = coll.Name
			}
			d.DescriptionText = coll.Description
			sanitizer := htmlsanitizer.NewHTMLSanitizer()
			sanitizedDesc, sanitizeErr := sanitizer.SanitizeString(d.DescriptionText)
			if sanitizeErr != nil {
				sanitizedDesc = template.HTMLEscapeString(d.DescriptionText)
			}
			d.DescriptionHTML = template.HTML(sanitizedDesc)
			d.BannerURL = coll.BannerURL
			d.Featured = coll.Featured
			if isAuthenticated(e) && coll.AuthorID != nil && *coll.AuthorID == authenticatedUserID(e) {
				d.IsOwner = true
			}

			// Load author name
			if coll.AuthorID != nil && *coll.AuthorID != "" {
				if u := findUserFromStore(appStore, *coll.AuthorID); u != nil {
					d.AuthorName = u.Username
				}
			}

			// Build the share URL
			scheme := "https"
			host := e.Request.Host
			if host == "" {
				host = "createmod.com"
			}
			if e.Request.TLS == nil {
				scheme = "http"
			}
			if d.Published {
				collSlug := coll.Slug
				if collSlug == "" {
					collSlug = coll.ID
				}
				d.ShareURL = fmt.Sprintf("%s://%s/collections/%s", scheme, host, collSlug)
			} else {
				d.ShareURL = fmt.Sprintf("%s://%s/collections/%s", scheme, host, coll.ID)
			}

			// Views increment with IP-based deduplication (1-hour window)
			clientIP := e.RealIP()
			ipKey := fmt.Sprintf("viewip:%s:coll:%s", clientIP, coll.ID)
			if clientIP != "" && cacheService != nil {
				if _, already := cacheService.Get(ipKey); !already {
					cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
					_ = appStore.Collections.IncrementViews(ctx, coll.ID)
					d.Views = coll.Views + 1
				} else {
					d.Views = coll.Views
				}
			} else {
				d.Views = coll.Views
			}

			// Load associated schematics
			schematicIDs, err := appStore.Collections.GetSchematicIDs(ctx, coll.ID)
			if err == nil && len(schematicIDs) > 0 {
				storeSchematics, err := appStore.Schematics.ListByIDs(ctx, schematicIDs)
				if err == nil {
					d.Schematics = MapStoreSchematics(appStore, storeSchematics, cacheService)
				}
			}
			d.SchematicCount = len(d.Schematics)
			translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)

			// Lazily generate collage for collections that have schematics but no banner or collage
			if coll.BannerURL == "" && coll.CollageURL == "" && len(schematicIDs) > 0 && storageSvc != nil {
				go generateCollectionCollage(storageSvc, appStore, coll.ID)
			}

			// Build mod info list from all schematics (deduplicated)
			seen := make(map[string]bool)
			var allMods []string
			for _, s := range d.Schematics {
				for _, mod := range s.Mods {
					if !seen[mod] {
						seen[mod] = true
						allMods = append(allMods, mod)
					}
				}
			}
			if len(allMods) > 0 {
				d.ModInfoList = buildModInfoListFromStore(appStore, allMods)
			}

			// Translation: show translated content if user's language is not English
			showOriginal := e.Request.URL.Query().Get("lang") == "original"
			if !showOriginal && translationService != nil && d.Language != "" && d.Language != "en" {
				t := translationService.GetCollectionTranslationCached(cacheService, coll.ID, d.Language)
				if t != nil && t.Title != "" {
					d.TitleText = t.Title
					if t.Description != "" {
						d.DescriptionText = t.Description
						sanitizedTransDesc, sanitizeErr := sanitizer.SanitizeString(t.Description)
						if sanitizeErr != nil {
							sanitizedTransDesc = template.HTMLEscapeString(t.Description)
						}
						d.DescriptionHTML = template.HTML(sanitizedTransDesc)
					}
					d.IsTranslated = true
				}
			}

			d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Collections"), "/collections", d.TitleText)

			// SEO/meta
			d.Title = d.TitleText
			if d.Title == "" {
				d.Title = i18n.T(d.Language, "Collection")
			}
			if d.DescriptionText != "" {
				d.Description = d.DescriptionText
			} else {
				d.Description = i18n.T(d.Language, "page.collections.description")
			}
			// og:image thumbnail
			if coll.BannerURL != "" {
				d.Thumbnail = "https://createmod.com" + coll.BannerURL
			} else if coll.CollageURL != "" {
				d.Thumbnail = "https://createmod.com" + coll.CollageURL
			}
		} else {
			// Not found
			d.Title = i18n.T(d.Language, "page.collections.notfound.title")
			d.Description = i18n.T(d.Language, "page.collections.notfound.description")
		}

		html, err := registry.LoadFiles(collectionsShowTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
