package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"createmod/internal/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// validModSlug matches mod namespace slugs: lowercase alphanumeric, underscores, hyphens, up to 128 chars.
var validModSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,127}$`)

var modsTemplates = append([]string{
	"./template/mods.html",
}, commonTemplates...)

var modDetailTemplates = append([]string{
	"./template/mod_detail.html",
	"./template/include/schematic_card.html",
}, commonTemplates...)

// ModEntry represents a single mod with its display info and schematic count.
type ModEntry struct {
	Slug          string
	Name          string
	Description   string
	IconURL       string
	ModrinthURL   string
	CurseForgeURL string
	Count         int
	IsVanilla     bool
}

// ModsListData holds the data for the mods listing page.
type ModsListData struct {
	DefaultData
	Mods      []ModEntry
	TotalMods int
}

// ModDetailData holds the data for a single mod's schematics page.
type ModDetailData struct {
	DefaultData
	Mod        ModEntry
	Subtitle   string
	Schematics []models.Schematic
	Page       int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	TotalCount int
}

const modsCacheKey = "mods_listing"

// ModsHandler renders the mods listing page at GET /mods.
func ModsHandler(cacheService *cache.Service, registry *server.Registry, modMetaService interface{}, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		mods, found := getCachedMods(cacheService)
		if !found {
			var err error
			mods, err = queryModEntriesFromStore(appStore)
			if err != nil {
				return err
			}
			enrichModEntriesFromStore(appStore, mods)
			cacheService.SetWithTTL(modsCacheKey, mods, 30*time.Minute)
		}

		d := ModsListData{
			Mods:      mods,
			TotalMods: len(mods),
		}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Mods"))
		d.Title = i18n.T(d.Language, "Mods")
		d.Description = i18n.T(d.Language, "Browse schematics by mod")
		d.Slug = "/mods"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(modsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// ModDetailHandler renders a specific mod's schematics at GET /mods/{slug}.
func ModDetailHandler(cacheService *cache.Service, registry *server.Registry, modMetaService interface{}, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/mods"))
		}
		if !validModSlug.MatchString(slug) {
			return e.NotFoundError("", nil)
		}

		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		limit := pageSize + 1
		offset := (page - 1) * pageSize

		ctx := context.Background()
		isVanilla := slug == "vanilla"
		var storeSchematics []store.Schematic
		var totalCount int
		var err error

		if isVanilla {
			storeSchematics, totalCount, err = appStore.Schematics.ListVanilla(ctx, limit, offset)
			if err != nil {
				return err
			}
		} else {
			// Check cache for total count (avoids expensive JSONB LATERAL COUNT on every request)
			countCacheKey := fmt.Sprintf("mod_count:%s", slug)
			if cached, ok := cacheService.Get(countCacheKey); ok {
				totalCount = cached.(int)
			} else {
				totalCount, err = appStore.Schematics.CountByMod(ctx, slug)
				if err != nil {
					return err
				}
				cacheService.SetWithTTL(countCacheKey, totalCount, 30*time.Minute)
			}
			// Fetch paginated schematics (the ID query is fast)
			storeSchematics, err = appStore.Schematics.ListByModPaginated(ctx, slug, limit, offset)
			if err != nil {
				return err
			}
		}

		hasNext := len(storeSchematics) > pageSize
		if hasNext {
			storeSchematics = storeSchematics[:pageSize]
		}

		caser := cases.Title(language.English)
		modName := caser.String(strings.ReplaceAll(slug, "_", " "))
		if isVanilla {
			modName = "Vanilla"
		}

		mod := ModEntry{
			Slug:      slug,
			Name:      modName,
			Count:     totalCount,
			IsVanilla: isVanilla,
		}

		// Enrich with metadata from store
		if !isVanilla {
			meta, mErr := appStore.ModMetadata.GetByNamespace(ctx, slug)
			if mErr == nil && meta != nil {
				if meta.DisplayName != "" {
					mod.Name = meta.DisplayName
				}
				mod.Description = meta.Description
				mod.IconURL = meta.IconURL
				mod.ModrinthURL = meta.ModrinthURL
				mod.CurseForgeURL = meta.CurseforgeURL
			}
		}

		d := ModDetailData{
			Mod:        mod,
			Schematics: MapStoreSchematics(appStore, storeSchematics, cacheService),
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
			TotalCount: totalCount,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/mods/%s?p=%d", slug, page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/mods/%s?p=%d", slug, page+1)
		}

		d.Populate(e)
		translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Mods"), "/mods", mod.Name)
		d.Title = mod.Name + " " + i18n.T(d.Language, "Schematics")
		if isVanilla {
			d.Subtitle = i18n.T(d.Language, "Schematics that require no mods in Minecraft")
			d.Description = d.Subtitle
		} else {
			d.Subtitle = fmt.Sprintf(i18n.T(d.Language, "Schematics using the %s mod in Minecraft"), mod.Name)
			d.Description = d.Subtitle
		}
		d.Slug = "/mods/" + slug
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(modDetailTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// getCachedMods retrieves the mod entries from cache if available.
func getCachedMods(cacheService *cache.Service) ([]ModEntry, bool) {
	val, found := cacheService.Get(modsCacheKey)
	if !found {
		return nil, false
	}
	mods, ok := val.([]ModEntry)
	if !ok {
		return nil, false
	}
	return mods, true
}

// queryModEntriesFromStore fetches all mods and their schematic counts from PostgreSQL.
func queryModEntriesFromStore(appStore *store.Store) ([]ModEntry, error) {
	ctx := context.Background()
	caser := cases.Title(language.English)

	modCounts, err := appStore.Schematics.ListModCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying mod entries: %w", err)
	}

	vanillaCount, err := appStore.Schematics.CountVanilla(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying vanilla count: %w", err)
	}

	entries := make([]ModEntry, 0, len(modCounts)+1)
	if vanillaCount > 0 {
		entries = append(entries, ModEntry{
			Slug:      "vanilla",
			Name:      "Vanilla",
			Count:     vanillaCount,
			IsVanilla: true,
		})
	}

	for _, mc := range modCounts {
		name := strings.TrimSpace(mc.ModName)
		if name == "" {
			continue
		}
		entries = append(entries, ModEntry{
			Slug:  strings.ToLower(strings.ReplaceAll(name, " ", "_")),
			Name:  caser.String(strings.ReplaceAll(name, "_", " ")),
			Count: mc.Count,
		})
	}

	return entries, nil
}

// enrichModEntriesFromStore populates metadata fields on mod entries from the store.
func enrichModEntriesFromStore(appStore *store.Store, entries []ModEntry) {
	ctx := context.Background()
	for i := range entries {
		if entries[i].IsVanilla {
			continue
		}
		meta, err := appStore.ModMetadata.GetByNamespace(ctx, entries[i].Slug)
		if err != nil || meta == nil {
			continue
		}
		if meta.DisplayName != "" {
			entries[i].Name = meta.DisplayName
		}
		entries[i].Description = meta.Description
		entries[i].IconURL = meta.IconURL
		entries[i].ModrinthURL = meta.ModrinthURL
		entries[i].CurseForgeURL = meta.CurseforgeURL
	}
}
