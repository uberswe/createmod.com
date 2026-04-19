package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"strings"
)

const fourOhFourTemplate = "./template/404.html"

var fourOhFourTemplates = append([]string{
	fourOhFourTemplate,
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_full.html",
}, commonTemplates...)

// FourOhFourData holds data for the 404 page, including similar schematics.
type FourOhFourData struct {
	DefaultData
	Similar []models.Schematic
}

func FourOhFourHandler(registry *server.Registry, searchEngine search.SearchEngine, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		return RenderNotFound(registry, searchEngine, cacheService, appStore, e)
	}
}

// RenderNotFound renders the 404 page with similar schematic recommendations
// derived from the request URL slug. It is safe to call from any handler.
func RenderNotFound(registry *server.Registry, searchEngine search.SearchEngine, cacheService *cache.Service, appStore *store.Store, e *server.RequestEvent) error {
	d := FourOhFourData{}
	d.Populate(e)
	d.Title = i18n.T(d.Language, "Page Not Found")
	d.Description = i18n.T(d.Language, "page.404.description")
	d.Slug = "/404"
	d.NoIndex = true
	d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

	// Extract search terms from the URL path slug
	terms := slugToSearchTerms(e.Request.URL.Path)
	if terms != "" && searchEngine != nil && searchEngine.Ready() {
		ids, err := searchEngine.Search(context.Background(), search.SearchQuery{Term: terms})
		if err == nil && len(ids) > 0 {
			if len(ids) > 6 {
				ids = ids[:6]
			}
			storeSchematics, err := appStore.Schematics.ListByIDs(context.Background(), ids)
			if err == nil && len(storeSchematics) > 0 {
				d.Similar = MapStoreSchematics(appStore, storeSchematics, cacheService)
			}
		} else if err != nil {
			slog.Warn("404 search failed", "terms", terms, "error", err)
		}
	}

	html, err := registry.LoadFiles(fourOhFourTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusNotFound, html)
}

// slugToSearchTerms extracts search terms from a URL path.
// For /schematics/steam-powered-house → "steam powered house".
// For other paths, uses the last path segment.
func slugToSearchTerms(path string) string {
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return ""
	}

	// Use the last path segment as the slug
	parts := strings.Split(path, "/")
	slug := parts[len(parts)-1]
	if slug == "" && len(parts) > 1 {
		slug = parts[len(parts)-2]
	}
	if slug == "" {
		return ""
	}

	// Split on hyphens and underscores, filter short words
	words := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_'
	})

	var filtered []string
	for _, w := range words {
		if len(w) >= 3 {
			filtered = append(filtered, w)
		}
	}

	return strings.Join(filtered, " ")
}
