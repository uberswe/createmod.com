package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
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
	"./template/include/schematic_card_list.html",
	"./template/include/mod_filters.html",
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
	TotalPages int

	// Filter fields
	Term                string
	ActiveTab           string // "trending", "rated", "latest"
	Category            string
	MinecraftVersion    string
	CreateVersion       string
	Rating              int
	MinBlockCount       int
	MaxBlockCount       int
	MinDimY             int
	MaxDimY             int
	MinHorizontal       int
	MaxHorizontal       int
	SearchResultCount   int
	SearchSpeed         string

	// Filter options
	MinecraftVersions   []models.MinecraftVersion
	CreateVersionGroups []CreateVersionGroup
	MaxBlockCountAll    int
	MaxDimYAll          int
	MaxHorizontalAll    int

	// View
	ViewMode       string
	PerPage        int
	InfiniteScroll bool
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
func ModDetailHandler(searchEngine search.SearchEngine, searchService *search.Service, cacheService *cache.Service, registry *server.Registry, modMetaService interface{}, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		start := time.Now()
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/mods"))
		}
		if !validModSlug.MatchString(slug) {
			return e.NotFoundError("", nil)
		}

		ctx := context.Background()
		isVanilla := slug == "vanilla"

		caser := cases.Title(language.English)
		modName := caser.String(strings.ReplaceAll(slug, "_", " "))
		if isVanilla {
			modName = "Vanilla"
		}

		mod := ModEntry{
			Slug:      slug,
			Name:      modName,
			IsVanilla: isVanilla,
		}

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

		q := e.Request.URL.Query()

		// Parse tab (determines sort order)
		activeTab := q.Get("tab")
		if activeTab != "rated" && activeTab != "latest" {
			activeTab = "trending"
		}
		var order int
		switch activeTab {
		case "rated":
			order = search.HighestRatingOrder
		case "latest":
			order = search.NewestOrder
		default:
			order = search.TrendingOrder
		}

		// Parse search term
		term := strings.TrimSpace(q.Get("q"))

		// Parse filters
		category := q.Get("category")
		if category == "" {
			category = "all"
		}
		mcVersion := q.Get("mcv")
		if mcVersion == "" {
			mcVersion = "all"
		}
		createVersion := q.Get("cv")
		if createVersion == "" {
			createVersion = "all"
		}

		rating := -1
		if q.Get("rating") != "" {
			if v, err := strconv.Atoi(q.Get("rating")); err == nil {
				rating = v
			}
		}

		parseIntParam := func(key string) int {
			v := q.Get(key)
			if v == "" {
				return 0
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return 0
			}
			return n
		}
		minBlockCount := parseIntParam("minbc")
		maxBlockCount := parseIntParam("maxbc")
		minDimY := parseIntParam("miny")
		maxDimY := parseIntParam("maxy")
		minHorizontal := parseIntParam("minhz")
		maxHorizontal := parseIntParam("maxhz")

		// Per page / infinite scroll
		infiniteScroll := q.Get("per_page") == "infinite"
		perPage := parseIntParam("per_page")
		if perPage != 8 && perPage != 16 && perPage != 32 && perPage != 64 {
			perPage = 0
		}
		viewMode := q.Get("view")
		if viewMode != "list" {
			viewMode = "grid"
		}

		// Resolve mod display name for Meilisearch filter
		var meiliModNames []string
		if !isVanilla {
			meiliModNames = []string{mod.Name}
		}

		// Expand major-version group selection
		var createVersionList []string
		if strings.HasPrefix(createVersion, "~") {
			prefix := strings.TrimPrefix(createVersion, "~")
			allCV := allCreatemodVersionsFromStore(appStore)
			for _, cv := range allCV {
				if createVersionMajor(cv.Version) == prefix {
					createVersionList = append(createVersionList, cv.Version)
				}
			}
		}

		sq := search.SearchQuery{
			Term:             term,
			Order:            order,
			Rating:           rating,
			Category:         category,
			MinecraftVersion: mcVersion,
			CreateVersion:    createVersion,
			CreateVersions:   createVersionList,
			MinBlockCount:    minBlockCount,
			MaxBlockCount:    maxBlockCount,
			MinDimY:          minDimY,
			MaxDimY:          maxDimY,
			MinHorizontal:    minHorizontal,
			MaxHorizontal:    maxHorizontal,
			Mods:             meiliModNames,
		}

		ids, _ := searchEngine.Search(e.Request.Context(), sq)

		storeSchematics, err := appStore.Schematics.ListByIDs(ctx, ids)
		if err != nil {
			return err
		}
		byID := make(map[string]store.Schematic, len(storeSchematics))
		for _, s := range storeSchematics {
			byID[s.ID] = s
		}
		orderedSchematics := make([]store.Schematic, 0, len(ids))
		for _, id := range ids {
			if s, ok := byID[id]; ok {
				if s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
					continue
				}
				orderedSchematics = append(orderedSchematics, s)
			}
		}

		// Pagination
		page := 1
		if p := q.Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 24
		if infiniteScroll {
			pageSize = 64
		} else if perPage > 0 {
			pageSize = perPage
		}
		total := len(orderedSchematics)
		maxPage := 0
		if pageSize > 0 {
			maxPage = (total + pageSize - 1) / pageSize
		}
		if maxPage > 0 && page > maxPage {
			page = maxPage
		}
		startIdx := (page - 1) * pageSize
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + pageSize
		if endIdx > total {
			endIdx = total
		}
		pageSlice := orderedSchematics
		if total > 0 {
			pageSlice = orderedSchematics[startIdx:endIdx]
		}

		// Build filter query string for pagination URLs
		queryParts := []string{}
		queryParts = append(queryParts, fmt.Sprintf("tab=%s", activeTab))
		if term != "" {
			queryParts = append(queryParts, fmt.Sprintf("q=%s", term))
		}
		if category != "all" {
			queryParts = append(queryParts, fmt.Sprintf("category=%s", category))
		}
		if mcVersion != "all" {
			queryParts = append(queryParts, fmt.Sprintf("mcv=%s", mcVersion))
		}
		if createVersion != "all" {
			queryParts = append(queryParts, fmt.Sprintf("cv=%s", createVersion))
		}
		if rating > 0 {
			queryParts = append(queryParts, fmt.Sprintf("rating=%d", rating))
		}
		if minBlockCount > 0 {
			queryParts = append(queryParts, fmt.Sprintf("minbc=%d", minBlockCount))
		}
		if maxBlockCount > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxbc=%d", maxBlockCount))
		}
		if minHorizontal > 0 {
			queryParts = append(queryParts, fmt.Sprintf("minhz=%d", minHorizontal))
		}
		if maxHorizontal > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxhz=%d", maxHorizontal))
		}
		if minDimY > 0 {
			queryParts = append(queryParts, fmt.Sprintf("miny=%d", minDimY))
		}
		if maxDimY > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxy=%d", maxDimY))
		}
		if infiniteScroll {
			queryParts = append(queryParts, "per_page=infinite")
		} else if perPage > 0 {
			queryParts = append(queryParts, fmt.Sprintf("per_page=%d", perPage))
		}
		if viewMode == "list" {
			queryParts = append(queryParts, "view=list")
		}

		buildPageURL := func(p int) string {
			parts := make([]string, 0, len(queryParts)+1)
			parts = append(parts, queryParts...)
			if p > 1 {
				parts = append(parts, fmt.Sprintf("p=%d", p))
			}
			qs := ""
			if len(parts) > 0 {
				qs = "?" + strings.Join(parts, "&")
			}
			return fmt.Sprintf("/mods/%s%s", slug, qs)
		}

		prevURL := ""
		nextURL := ""
		if page > 1 {
			prevURL = buildPageURL(page - 1)
		}
		if endIdx < total {
			nextURL = buildPageURL(page + 1)
		}

		totalPages := 0
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
		}

		maxStats := searchService.MaxStats()
		cvAll := allCreatemodVersionsFromStore(appStore)

		duration := time.Since(start)
		d := ModDetailData{
			Mod:               mod,
			Schematics:        MapStoreSchematics(appStore, pageSlice, cacheService),
			Page:              page,
			HasPrev:           prevURL != "",
			HasNext:           nextURL != "",
			PrevURL:           prevURL,
			NextURL:           nextURL,
			TotalCount:        total,
			TotalPages:        totalPages,
			Term:              term,
			ActiveTab:         activeTab,
			Category:          category,
			MinecraftVersion:  mcVersion,
			CreateVersion:     createVersion,
			Rating:            rating,
			MinBlockCount:     minBlockCount,
			MaxBlockCount:     maxBlockCount,
			MinDimY:           minDimY,
			MaxDimY:           maxDimY,
			MinHorizontal:     minHorizontal,
			MaxHorizontal:     maxHorizontal,
			SearchResultCount: total,
			SearchSpeed:       fmt.Sprintf("%.6f", duration.Seconds()),
			MinecraftVersions:   allMinecraftVersionsFromStore(appStore),
			CreateVersionGroups: groupCreateVersions(cvAll),
			MaxBlockCountAll:    maxStats.BlockCount,
			MaxDimYAll:          maxStats.DimY,
			MaxHorizontalAll:    max(maxStats.DimX, maxStats.DimZ),
			ViewMode:            viewMode,
			PerPage:             pageSize,
			InfiniteScroll:      infiniteScroll,
		}
		mod.Count = total

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
