package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"createmod/internal/translation"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"createmod/internal/server"
)

var exploreTemplates = append([]string{
	"./template/explore.html",
}, commonTemplates...)

type ExploreImage struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Title            string `json:"title"`
	Image            string `json:"image"`
	Size             string `json:"size"` // "sm", "md", or "lg"
	DetectedLanguage string `json:"-"`
}

type ExploreData struct {
	DefaultData
	Images []ExploreImage
	Seed   int64
}

const (
	explorePageSize      = 30
	exploreCacheKey      = "explore:images"
	exploreCacheDuration = 10 * time.Minute
)

// buildExploreImages fetches all approved schematics and flattens their images
// into an ExploreImage slice. The result is cached for exploreCacheDuration.
func buildExploreImages(appStore *store.Store, cacheService *cache.Service) ([]ExploreImage, error) {
	if cached, ok := cacheService.Get(exploreCacheKey); ok {
		if images, ok := cached.([]ExploreImage); ok {
			return images, nil
		}
	}

	storeSchematics, err := appStore.Schematics.ListAllForIndex(context.Background())
	if err != nil {
		return nil, err
	}

	images := make([]ExploreImage, 0, len(storeSchematics)*2)
	for _, s := range storeSchematics {
		if s.FeaturedImage == "" || s.Name == "" {
			continue
		}
		for _, g := range s.Gallery {
			if g == "" {
				continue
			}
			images = append(images, ExploreImage{
				ID:               s.ID,
				Title:            s.Title,
				Name:             s.Name,
				Image:            g,
				DetectedLanguage: s.DetectedLanguage,
			})
		}
		images = append(images, ExploreImage{
			ID:               s.ID,
			Title:            s.Title,
			Name:             s.Name,
			Image:            s.FeaturedImage,
			DetectedLanguage: s.DetectedLanguage,
		})
	}

	cacheService.SetWithTTL(exploreCacheKey, images, exploreCacheDuration)
	return images, nil
}

// translateExploreImageTitles replaces each explore image's title with its cached
// translation when the viewer's language differs from the image's detected language.
func translateExploreImageTitles(images []ExploreImage, translationService *translation.Service, cacheService *cache.Service, targetLang string) {
	if translationService == nil || cacheService == nil || targetLang == "" {
		return
	}
	for i := range images {
		detectedLang := images[i].DetectedLanguage
		if detectedLang == "" {
			detectedLang = "en"
		}
		if detectedLang == targetLang {
			continue
		}
		t := translationService.GetTranslationCached(cacheService, images[i].ID, targetLang)
		if t != nil && t.Title != "" {
			images[i].Title = t.Title
		}
	}
}

// shuffleImages returns a deterministically shuffled copy of images using the
// given seed, with random sizes assigned.
func shuffleImages(images []ExploreImage, seed int64) []ExploreImage {
	n := len(images)
	if n == 0 {
		return nil
	}
	dest := make([]ExploreImage, n)
	copy(dest, images)

	// Deterministic shuffle using the seed
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed>>1|1)))
	rng.Shuffle(n, func(i, j int) {
		dest[i], dest[j] = dest[j], dest[i]
	})

	// Assign varied sizes for visual interest in the masonry grid.
	// Roughly 15% large, 35% medium, 50% small.
	for i := range dest {
		r := rng.IntN(100)
		switch {
		case r < 15:
			dest[i].Size = "lg"
		case r < 50:
			dest[i].Size = "md"
		default:
			dest[i].Size = "sm"
		}
	}

	return dest
}

func ExploreHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		images, err := buildExploreImages(appStore, cacheService)
		if err != nil {
			return err
		}

		seed := time.Now().UnixNano()
		shuffled := shuffleImages(images, seed)

		// Take only the first page for initial render
		show := shuffled
		if len(show) > explorePageSize {
			show = show[:explorePageSize]
		}

		d := ExploreData{
			Images: show,
			Seed:   seed,
		}
		d.Populate(e)
		translateExploreImageTitles(d.Images, translationService, cacheService, d.Language)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "page.explore.title"))
		d.Title = i18n.T(d.Language, "page.explore.title")
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Description = i18n.T(d.Language, "page.explore.description")
		d.Slug = "/explore"
		if len(show) > 0 {
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", show[0].ID, url.PathEscape(show[0].Image))
		}
		html, err := registry.LoadFiles(exploreTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// ExploreAPIHandler serves GET /api/explore/images?seed=X&cursor=N&limit=30
// Returns a JSON batch of shuffled explore images for infinite scroll.
func ExploreAPIHandler(cacheService *cache.Service, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		images, err := buildExploreImages(appStore, cacheService)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to load images"})
		}
		if len(images) == 0 {
			return writeJSON(e, http.StatusOK, map[string]interface{}{
				"images":   []ExploreImage{},
				"cursor":   0,
				"seed":     0,
				"has_more": false,
			})
		}

		q := e.Request.URL.Query()

		seed, _ := strconv.ParseInt(q.Get("seed"), 10, 64)
		if seed == 0 {
			seed = time.Now().UnixNano()
		}

		cursor, _ := strconv.Atoi(q.Get("cursor"))
		if cursor < 0 {
			cursor = 0
		}

		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 || limit > 120 {
			limit = explorePageSize
		}

		shuffled := shuffleImages(images, seed)
		total := len(shuffled)

		// If cursor exceeds the total, start a new cycle with a new seed
		newSeed := seed
		if cursor >= total {
			cursor = 0
			newSeed = seed + 1
			shuffled = shuffleImages(images, newSeed)
		}

		end := cursor + limit
		if end > total {
			end = total
		}
		batch := shuffled[cursor:end]
		translateExploreImageTitles(batch, translationService, cacheService, detectLanguageFromRequest(e.Request))

		resp := map[string]interface{}{
			"images":   batch,
			"cursor":   end,
			"seed":     newSeed,
			"total":    total,
			"has_more": true, // always true — infinite scroll wraps around
		}

		data, _ := json.Marshal(resp)
		e.Response.Header().Set("Content-Type", "application/json")
		return e.String(http.StatusOK, string(data))
	}
}
