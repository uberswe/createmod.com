package pages

import (
	"context"
	"createmod/internal/cache"
	"encoding/json"
	"createmod/internal/i18n"
	"createmod/internal/metrics"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"createmod/internal/server"
)

var searchTemplates = append([]string{
	"./template/search.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_list.html",
	"./template/include/schematic_card_medium.html",
	"./template/include/search_filters.html",
	"./template/include/search_pagination.html",
}, commonTemplates...)

// ModOption represents a mod entry for the search filter multiselect.
type ModOption struct {
	Namespace   string // raw mod namespace for URL params (e.g. "create")
	DisplayName string // human-readable name (e.g. "Create")
	Count       int
}

// CreateVersionGroup holds a major version group label and its child versions.
type CreateVersionGroup struct {
	Label    string                   // e.g. "6.0.x", "0.5.x"
	Value    string                   // e.g. "~6.0", "~0.5"  (prefix marker)
	Versions []models.CreatemodVersion // individual versions
}

type SearchData struct {
	DefaultData
	// HeadingText overrides the visible h1 on category/tag landing pages.
	HeadingText          string
	Schematics           []models.Schematic
	Tags                 []models.SchematicTag
	TagsWithCount        []models.SchematicTagWithCount
	SelectedTags         []string
	MinecraftVersions    []models.MinecraftVersion
	CreateVersions       []models.CreatemodVersion
	CreateVersionGroups  []CreateVersionGroup
	SearchSpeed          string
	SearchResultCount    int // total results count
	TotalResults         int
	TotalPages           int
	PageNumbers          []int // sliding window page numbers; -1 = ellipsis
	Term                 string
	TermSlug             string
	Sort                 int
	DisplaySort          int // sort value shown in the UI (always BestMatch when no explicit sort)
	Rating               int
	Category             string
	Tag                  string // backward compat: first selected tag
	MinecraftVersion     string
	CreateVersion        string
	CreateVersionDisplay string
	Page                 int
	PageSize             int
	HasPrev              bool
	HasNext              bool
	PrevURL              string
	NextURL              string
	FirstURL             string
	LastURL              string
	ViewMode             string
	MinBlockCount        int
	MaxBlockCount        int
	MinDimX              int
	MaxDimX              int
	MinDimY              int
	MaxDimY              int
	MinDimZ              int
	MaxDimZ              int
	MinHorizontal        int
	MaxHorizontal        int
	SelectedMods         []string
	AllMods              []ModOption
	ModMatch             string
	InfiniteScroll       bool
	MaxBlockCountAll     int // global max for slider upper bound
	MaxDimXAll           int
	MaxDimYAll           int
	MaxDimZAll           int
	MaxHorizontalAll     int
	PerPage              int
}

// groupCreateVersions groups a flat list of Create mod versions into major version groups.
// Major versions: anything starting with "6." → "6.0", otherwise take first two segments (e.g. "0.5").
func groupCreateVersions(versions []models.CreatemodVersion) []CreateVersionGroup {
	type entry struct {
		key      string
		label    string
		versions []models.CreatemodVersion
	}
	var order []string
	groups := map[string]*entry{}

	for _, v := range versions {
		ver := v.Version
		var major string
		if strings.HasPrefix(ver, "6.") || strings.HasPrefix(ver, "6 ") {
			major = "6.0"
		} else {
			parts := strings.SplitN(ver, ".", 3)
			if len(parts) >= 2 {
				minor := parts[1]
				// Extract just the leading digits from the minor part
				// so "4c"→"4", "31a"→"3" (first digit only for legacy naming)
				if len(minor) > 0 && minor[0] >= '0' && minor[0] <= '9' {
					minor = string(minor[0])
				}
				major = parts[0] + "." + minor
			} else {
				// Single-segment version like "0.31a" without dots
				major = "0.x"
			}
		}
		e, ok := groups[major]
		if !ok {
			e = &entry{key: major, label: major + ".x"}
			groups[major] = e
			order = append(order, major)
		}
		e.versions = append(e.versions, v)
	}

	result := make([]CreateVersionGroup, 0, len(order))
	for _, key := range order {
		e := groups[key]
		result = append(result, CreateVersionGroup{
			Label:    e.label,
			Value:    "~" + e.key,
			Versions: e.versions,
		})
	}
	return result
}

func createVersionDisplay(cv string) string {
	if strings.HasPrefix(cv, "~") {
		return strings.TrimPrefix(cv, "~") + ".x"
	}
	return cv
}

func createVersionMajor(ver string) string {
	if strings.HasPrefix(ver, "6.") || strings.HasPrefix(ver, "6 ") {
		return "6.0"
	}
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) >= 2 {
		minor := parts[1]
		if len(minor) > 0 && minor[0] >= '0' && minor[0] <= '9' {
			minor = string(minor[0])
		}
		return parts[0] + "." + minor
	}
	return "0.x"
}

func SearchHandler(searchEngine search.SearchEngine, searchService *search.Service, cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		start := time.Now()
		slugTerm := e.Request.PathValue("term")
		// Also accept ?q= query param (used by live HTMX filtering)
		if slugTerm == "" {
			if q := strings.TrimSpace(e.Request.URL.Query().Get("q")); q != "" {
				slugTerm = slug.Make(q)
			}
		}

		// Default sort: BestMatch in the UI; actual query uses Trending when
		// browsing without a search term so results are meaningful.
		hasSortParam := e.Request.URL.Query().Get("sort") != ""
		order := search.BestMatchOrder
		displaySort := search.BestMatchOrder
		if slugTerm == "" && !hasSortParam {
			// No search term and no explicit sort: query by trending but
			// show "Best match" selected in the UI.
			order = search.TrendingOrder
		}
		if hasSortParam {
			atoi, err := strconv.Atoi(e.Request.URL.Query().Get("sort"))
			if err != nil {
				return err
			}
			order = atoi
			displaySort = atoi
		}

		rating := -1
		if e.Request.URL.Query().Get("rating") != "" {
			atoi, err := strconv.Atoi(e.Request.URL.Query().Get("rating"))
			if err != nil {
				return err
			}
			rating = atoi
		}
		category := "all"
		if e.Request.URL.Query().Get("category") != "" {
			category = e.Request.URL.Query().Get("category")
		}
		// Multi-tag: parse comma-separated tag param
		tagParam := e.Request.URL.Query().Get("tag")
		var selectedTags []string
		if tagParam != "" && tagParam != "all" {
			for _, t := range strings.Split(tagParam, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					selectedTags = append(selectedTags, t)
				}
			}
		}
		// Build tags slice for search service (nil means no filter)
		searchTags := selectedTags
		if len(searchTags) == 0 {
			searchTags = nil
		}

		mcVersion := "all"
		if e.Request.URL.Query().Get("mcv") != "" {
			mcVersion = e.Request.URL.Query().Get("mcv")
		}
		createVersion := "all"
		if e.Request.URL.Query().Get("cv") != "" {
			createVersion = e.Request.URL.Query().Get("cv")
		}

		// Parse dimension and block count range filters
		parseIntParam := func(key string) int {
			v := e.Request.URL.Query().Get(key)
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
		minDimX := parseIntParam("minx")
		maxDimX := parseIntParam("maxx")
		minDimY := parseIntParam("miny")
		maxDimY := parseIntParam("maxy")
		minDimZ := parseIntParam("minz")
		maxDimZ := parseIntParam("maxz")
		minHorizontal := parseIntParam("minhz")
		maxHorizontal := parseIntParam("maxhz")

		// Parse per_page (query param overrides cookie)
		perPageRaw := e.Request.URL.Query().Get("per_page")
		if perPageRaw == "" {
			if c, err := e.Request.Cookie("cm_per_page_search"); err == nil {
				perPageRaw = c.Value
			}
		}
		infiniteScroll := perPageRaw == "infinite"
		perPage := 0
		if v, err := strconv.Atoi(perPageRaw); err == nil {
			perPage = v
		}
		if perPage != 8 && perPage != 16 && perPage != 32 && perPage != 64 {
			perPage = 0
		}

		// Parse mod filter (comma-separated "mods" param or multiple "mod" checkbox params)
		var selectedMods []string
		if modsParam := e.Request.URL.Query().Get("mods"); modsParam != "" {
			for _, m := range strings.Split(modsParam, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					selectedMods = append(selectedMods, m)
				}
			}
		}
		if len(selectedMods) == 0 {
			if modValues := e.Request.URL.Query()["mod"]; len(modValues) > 0 {
				for _, m := range modValues {
					m = strings.TrimSpace(m)
					if m != "" {
						selectedMods = append(selectedMods, m)
					}
				}
			}
		}

		modMatch := e.Request.URL.Query().Get("mod_match")
		if modMatch != "all" {
			modMatch = "any"
		}

		// Build mod options list and resolve selected namespaces to display names for Meilisearch
		allMods := allModOptionsFromStore(appStore, cacheService)
		maxStats := searchService.MaxStats()
		var meiliModNames []string
		if len(selectedMods) > 0 {
			nsToDisplay := make(map[string]string, len(allMods))
			for _, mo := range allMods {
				nsToDisplay[mo.Namespace] = mo.DisplayName
			}
			for _, ns := range selectedMods {
				if dn, ok := nsToDisplay[ns]; ok {
					meiliModNames = append(meiliModNames, dn)
				}
			}
		}

		term := strings.ReplaceAll(slugTerm, "-", " ")

		// Expand major-version group selection (e.g. "~6.0") into individual versions.
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
			Tags:             searchTags,
			MinecraftVersion: mcVersion,
			CreateVersion:    createVersion,
			CreateVersions:   createVersionList,
			MinBlockCount:    minBlockCount,
			MaxBlockCount:    maxBlockCount,
			MinDimX:          minDimX,
			MaxDimX:          maxDimX,
			MinDimY:          minDimY,
			MaxDimY:          maxDimY,
			MinDimZ:          minDimZ,
			MaxDimZ:          maxDimZ,
			MinHorizontal:    minHorizontal,
			MaxHorizontal:    maxHorizontal,
			Mods:             meiliModNames,
		}

		ids, _ := searchEngine.Search(e.Request.Context(), sq)

		// Record metrics.
		searchDuration := time.Since(start)
		metrics.SearchLatency.WithLabelValues("meilisearch", "mods").Observe(searchDuration.Seconds())
		zeroResults := "false"
		if len(ids) == 0 {
			zeroResults = "true"
		}
		metrics.SearchQueries.WithLabelValues("meilisearch", "mods", zeroResults).Inc()

		slog.Info("search",
			"event", "search",
			"engine", "meilisearch",
			"index", "mods",
			"query", term,
			"results_count", len(ids),
			"zero_results", len(ids) == 0,
			"latency_ms", searchDuration.Milliseconds(),
		)

		// Fetch schematics from store by IDs
		ctx := context.Background()
		storeSchematics, err := appStore.Schematics.ListByIDs(ctx, ids)
		if err != nil {
			return err
		}
		// Build a lookup map for ordering
		byID := make(map[string]store.Schematic, len(storeSchematics))
		for _, s := range storeSchematics {
			byID[s.ID] = s
		}
		// Preserve search result order and filter to approved only
		orderedSchematics := make([]store.Schematic, 0, len(ids))
		for _, id := range ids {
			if s, ok := byID[id]; ok {
				if s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
					continue
				}
				orderedSchematics = append(orderedSchematics, s)
			}
		}

		// "Has only these mods" post-filter: exclude schematics with mods not in the selected set
		if modMatch == "all" && len(selectedMods) > 0 {
			allowedSet := make(map[string]bool, len(meiliModNames))
			for _, dn := range meiliModNames {
				allowedSet[dn] = true
			}
			filtered := orderedSchematics[:0]
			for _, s := range orderedSchematics {
				if s.Mods == nil {
					continue
				}
				var sMods []string
				if err := json.Unmarshal(s.Mods, &sMods); err != nil {
					continue
				}
				onlySelected := true
				for _, m := range sMods {
					if !allowedSet[m] {
						onlySelected = false
						break
					}
				}
				if onlySelected {
					filtered = append(filtered, s)
				}
			}
			orderedSchematics = filtered
		}

		// Pagination: check path value first, fall back to ?p= query param
		page := 1
		if pathPage := e.Request.PathValue("page"); pathPage != "" {
			if p, err := strconv.Atoi(pathPage); err == nil && p > 0 {
				page = p
			}
		} else if e.Request.URL.Query().Get("p") != "" {
			if p, err := strconv.Atoi(e.Request.URL.Query().Get("p")); err == nil && p > 0 {
				page = p
			}
		}
		pageSize := 8
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

		schematicModels := MapStoreSchematics(appStore, pageSlice, cacheService)

		end := time.Now()
		duration := end.Sub(start)

		// Reconstruct tag param for URLs
		tagURLParam := "all"
		if len(selectedTags) > 0 {
			tagURLParam = strings.Join(selectedTags, ",")
		}

		// Build query string for filter params — only include sort when explicitly set
		// so the smart default logic applies consistently across pagination
		queryParts := []string{}
		if hasSortParam {
			queryParts = append(queryParts, fmt.Sprintf("sort=%d", order))
		}
		queryParts = append(queryParts, fmt.Sprintf("rating=%d", rating))
		queryParts = append(queryParts, fmt.Sprintf("category=%s", category))
		queryParts = append(queryParts, fmt.Sprintf("tag=%s", tagURLParam))
		queryParts = append(queryParts, fmt.Sprintf("mcv=%s", mcVersion))
		queryParts = append(queryParts, fmt.Sprintf("cv=%s", createVersion))
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
		if infiniteScroll {
			queryParts = append(queryParts, "per_page=infinite")
		} else if perPage > 0 {
			queryParts = append(queryParts, fmt.Sprintf("per_page=%d", perPage))
		}
		if minDimX > 0 {
			queryParts = append(queryParts, fmt.Sprintf("minx=%d", minDimX))
		}
		if maxDimX > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxx=%d", maxDimX))
		}
		if minDimY > 0 {
			queryParts = append(queryParts, fmt.Sprintf("miny=%d", minDimY))
		}
		if maxDimY > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxy=%d", maxDimY))
		}
		if minDimZ > 0 {
			queryParts = append(queryParts, fmt.Sprintf("minz=%d", minDimZ))
		}
		if maxDimZ > 0 {
			queryParts = append(queryParts, fmt.Sprintf("maxz=%d", maxDimZ))
		}
		if len(selectedMods) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("mods=%s", strings.Join(selectedMods, ",")))
			if modMatch == "all" {
				queryParts = append(queryParts, "mod_match=all")
			}
		}

		viewMode := e.Request.URL.Query().Get("view")
		if viewMode != "list" {
			viewMode = "grid"
		}
		if viewMode == "list" {
			queryParts = append(queryParts, "view=list")
		}

		// Build query-param pagination URLs (works with both path-based and HTMX ?q= requests)
		buildPageURL := func(p int) string {
			parts := make([]string, 0, len(queryParts)+2)
			if slugTerm != "" {
				parts = append(parts, fmt.Sprintf("q=%s", slugTerm))
			}
			parts = append(parts, queryParts...)
			if p > 1 {
				parts = append(parts, fmt.Sprintf("p=%d", p))
			}
			return fmt.Sprintf("/search?%s", strings.Join(parts, "&"))
		}

		prevURL := ""
		nextURL := ""
		if page > 1 {
			prevURL = buildPageURL(page - 1)
		}
		if endIdx < total {
			nextURL = buildPageURL(page + 1)
		}

		// SEO classification: clean landings (bare search, single category,
		// single tag) are indexable and self-canonical; every other filter
		// permutation is noindexed and canonicalizes to its dominant clean
		// landing so ranking signals consolidate instead of fragmenting
		// across facet combinations.
		//
		// Pages with a user-supplied search term are ALWAYS noindexed: the
		// term space is unbounded attacker-controlled input, and spammers
		// link junk queries (e.g. "<brand>-premium-mod-for-pc") from
		// external sites to get them indexed, which attracts brand-protection
		// crawlers and DMCA notices against auto-generated result pages.
		// Category and tag landings stay indexable because those vocabularies
		// are site-controlled.
		hasExtraFilters := rating > -1 || mcVersion != "all" || createVersion != "all" ||
			minBlockCount > 0 || maxBlockCount > 0 || minHorizontal > 0 || maxHorizontal > 0 ||
			minDimX > 0 || maxDimX > 0 || minDimY > 0 || maxDimY > 0 || minDimZ > 0 || maxDimZ > 0 ||
			len(selectedMods) > 0 || hasSortParam || infiniteScroll || viewMode == "list" ||
			e.Request.URL.Query().Get("per_page") != ""

		isCleanTerm := slugTerm != "" && category == "all" && len(selectedTags) == 0 && !hasExtraFilters
		isCleanCategory := slugTerm == "" && category != "all" && len(selectedTags) == 0 && !hasExtraFilters
		isCleanTag := slugTerm == "" && category == "all" && len(selectedTags) == 1 && !hasExtraFilters
		isCleanBare := slugTerm == "" && category == "all" && len(selectedTags) == 0 && !hasExtraFilters
		isCleanLanding := isCleanTerm || isCleanCategory || isCleanTag || isCleanBare

		canonicalPath := "/search"
		switch {
		case slugTerm != "":
			canonicalPath = "/search?q=" + url.QueryEscape(slugTerm)
		case category != "all":
			canonicalPath = "/search?category=" + url.QueryEscape(category)
		case len(selectedTags) > 0:
			canonicalPath = "/search?tag=" + url.QueryEscape(selectedTags[0])
		}
		if isCleanLanding && page > 1 {
			sep := "?"
			if strings.Contains(canonicalPath, "?") {
				sep = "&"
			}
			canonicalPath += fmt.Sprintf("%sp=%d", sep, page)
		}
		seoNoIndex := page > 20 || !isCleanLanding || slugTerm != ""

		totalPages := 0
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
		}

		cvAll := allCreatemodVersionsFromStore(appStore)
		d := SearchData{
			Schematics:          schematicModels,
			Tags:                allTagsFromStore(appStore),
			TagsWithCount:       allTagsWithCountFromStore(appStore, cacheService),
			SelectedTags:        selectedTags,
			MinecraftVersions:   allMinecraftVersionsFromStore(appStore),
			CreateVersions:      cvAll,
			CreateVersionGroups: groupCreateVersions(cvAll),
			SearchSpeed:       fmt.Sprintf("%.6f", duration.Seconds()),
			SearchResultCount: total,
			TotalResults:      total,
			TotalPages:        totalPages,
			PageNumbers:       computePageNumbers(page, totalPages),
			Term:              term,
			TermSlug:          slugTerm,
			Sort:              order,
			DisplaySort:       displaySort,
			Rating:            rating,
			Category:          category,
			Tag:               tagURLParam,
			MinecraftVersion:     mcVersion,
			CreateVersion:        createVersion,
			CreateVersionDisplay: createVersionDisplay(createVersion),
			Page:                 page,
			PageSize:          pageSize,
			HasPrev:           prevURL != "",
			HasNext:           nextURL != "",
			PrevURL:           prevURL,
			NextURL:           nextURL,
			FirstURL:          buildPageURL(1),
			LastURL:           func() string { if totalPages > 0 { return buildPageURL(totalPages) }; return "" }(),
			ViewMode:          viewMode,
			MinBlockCount:     minBlockCount,
			MaxBlockCount:     maxBlockCount,
			MinDimX:           minDimX,
			MaxDimX:           maxDimX,
			MinDimY:           minDimY,
			MaxDimY:           maxDimY,
			MinDimZ:           minDimZ,
			MaxDimZ:           maxDimZ,
			MinHorizontal:     minHorizontal,
			MaxHorizontal:     maxHorizontal,
			SelectedMods:      selectedMods,
			AllMods:           allMods,
			ModMatch:          modMatch,
			MaxBlockCountAll:  maxStats.BlockCount,
			MaxDimXAll:        maxStats.DimX,
			MaxDimYAll:        maxStats.DimY,
			MaxDimZAll:        maxStats.DimZ,
			MaxHorizontalAll:  max(maxStats.DimX, maxStats.DimZ),
			InfiniteScroll:    infiniteScroll,
			PerPage:           pageSize,
		}
		d.Populate(e)
		translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Search"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Dynamic title, heading and description based on search context.
		// Category and tag landings get keyword-rich copy so they can rank as
		// browse pages instead of generic "Search" results.
		catName := ""
		if category != "all" {
			for _, c := range d.Categories {
				if c.Key == category {
					catName = i18n.T(d.Language, c.Name)
					break
				}
			}
		}
		tagName := ""
		if len(selectedTags) == 1 {
			for _, t := range d.Tags {
				if t.Key == selectedTags[0] {
					tagName = i18n.T(d.Language, t.Name)
					break
				}
			}
		}
		switch {
		case isCleanCategory && catName != "":
			d.Title = fmt.Sprintf("%s %s", catName, i18n.T(d.Language, "Create Mod Schematics"))
			d.HeadingText = d.Title
			d.Description = truncateMetaDescription(fmt.Sprintf(i18n.T(d.Language, "Browse %d %s schematics for the Minecraft Create Mod. Download them for your world or share your own builds."), total, catName))
		case isCleanTag && tagName != "":
			d.Title = fmt.Sprintf("%s %s", tagName, i18n.T(d.Language, "Create Mod Schematics"))
			d.HeadingText = d.Title
			d.Description = truncateMetaDescription(fmt.Sprintf(i18n.T(d.Language, "Browse %d %s schematics for the Minecraft Create Mod. Download them for your world or share your own builds."), total, tagName))
		case term != "":
			d.Title = fmt.Sprintf("%s "+i18n.T(d.Language, "Schematics"), term)
			d.Description = fmt.Sprintf(i18n.T(d.Language, "page.search.description"), d.Term)
		default:
			d.Title = i18n.T(d.Language, "Search")
			d.Description = fmt.Sprintf(i18n.T(d.Language, "page.search.description"), d.Term)
		}
		if page > 1 {
			d.Title = fmt.Sprintf("%s - %s %d", d.Title, i18n.T(d.Language, "Page"), page)
		}
		d.Slug = canonicalPath
		d.CanonicalURL = fmt.Sprintf("https://createmod.com%s", PrefixedPath(d.Language, canonicalPath))
		d.NoIndex = seoNoIndex
		if prevURL != "" {
			d.PrevPageURL = fmt.Sprintf("https://createmod.com%s", PrefixedPath(d.Language, prevURL))
		}
		if nextURL != "" {
			d.NextPageURL = fmt.Sprintf("https://createmod.com%s", PrefixedPath(d.Language, nextURL))
		}
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		if d.SearchResultCount > 0 {
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematics[0].ID, url.PathEscape(d.Schematics[0].FeaturedImage))
		}

		html, err := registry.LoadFiles(searchTemplates...).Render(d)
		if err != nil {
			return err
		}
		if len(term) >= 3 {
			_ = appStore.SearchTracking.RecordSearch(ctx, term, d.SearchResultCount, authenticatedUserID(e), e.RealIP())
		}
		return e.HTML(http.StatusOK, html)
	}
}

// computePageNumbers returns a sliding window of page numbers with ellipsis markers (-1).
func computePageNumbers(current, totalPages int) []int {
	if totalPages <= 1 {
		return nil
	}
	const windowSize = 5
	pages := make([]int, 0, windowSize+4)

	start := current - windowSize/2
	end := current + windowSize/2

	if start < 1 {
		start = 1
		end = windowSize
	}
	if end > totalPages {
		end = totalPages
		start = totalPages - windowSize + 1
	}
	if start < 1 {
		start = 1
	}

	if start > 1 {
		pages = append(pages, 1)
		if start > 2 {
			pages = append(pages, -1) // ellipsis
		}
	}
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	if end < totalPages {
		if end < totalPages-1 {
			pages = append(pages, -1) // ellipsis
		}
		pages = append(pages, totalPages)
	}
	return pages
}

// searchCount is no longer used — search tracking is done via appStore.SearchTracking.RecordSearch

func SearchPostHandler(service *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		data := struct {
			Term             string `json:"q" form:"q"`
			Sort             string `json:"sort" form:"sort"`
			Rating           string `json:"rating" form:"rating"`
			Category         string `json:"category" form:"category"`
			Tag              string `json:"tag" form:"tag"`
			MinecraftVersion string `json:"mcv" form:"mcv"`
			CreateVersion    string `json:"cv" form:"cv"`
			MinBlockCount    string `json:"minbc" form:"minbc"`
			MaxBlockCount    string `json:"maxbc" form:"maxbc"`
			MinHorizontal    string `json:"minhz" form:"minhz"`
			MaxHorizontal    string `json:"maxhz" form:"maxhz"`
			MinDimY          string `json:"miny" form:"miny"`
			MaxDimY          string `json:"maxy" form:"maxy"`
			PerPage          string `json:"per_page" form:"per_page"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return &server.APIError{Status: 400, Message: "Failed to read request data"}
		}
		// Handle multi-tag: form may submit multiple tag values via checkboxes
		tagParam := data.Tag
		if tagParam == "" {
			// Check for multiple tag values from checkboxes
			if err := e.Request.ParseForm(); err == nil {
				if tagValues := e.Request.Form["tag"]; len(tagValues) > 1 {
					tagParam = strings.Join(tagValues, ",")
				}
			}
		}
		if tagParam == "" {
			tagParam = "all"
		}
		// Handle multi-mod: form may submit multiple mod values via checkboxes
		var modsParam string
		if err := e.Request.ParseForm(); err == nil {
			if modValues := e.Request.Form["mod"]; len(modValues) > 0 {
				modsParam = strings.Join(modValues, ",")
			}
		}
		term := slug.Make(data.Term)
		redirectURL := fmt.Sprintf("/search/%s?sort=%s&rating=%s&category=%s&tag=%s&mcv=%s&cv=%s", term, data.Sort, data.Rating, data.Category, tagParam, data.MinecraftVersion, data.CreateVersion)
		if data.MinBlockCount != "" && data.MinBlockCount != "0" {
			redirectURL += "&minbc=" + data.MinBlockCount
		}
		if data.MaxBlockCount != "" && data.MaxBlockCount != "0" {
			redirectURL += "&maxbc=" + data.MaxBlockCount
		}
		if data.MinHorizontal != "" && data.MinHorizontal != "0" {
			redirectURL += "&minhz=" + data.MinHorizontal
		}
		if data.MaxHorizontal != "" && data.MaxHorizontal != "0" {
			redirectURL += "&maxhz=" + data.MaxHorizontal
		}
		if data.PerPage != "" && data.PerPage != "0" && data.PerPage != "18" {
			redirectURL += "&per_page=" + data.PerPage
		}
		if data.MinDimY != "" && data.MinDimY != "0" {
			redirectURL += "&miny=" + data.MinDimY
		}
		if data.MaxDimY != "" && data.MaxDimY != "0" {
			redirectURL += "&maxy=" + data.MaxDimY
		}
		if modsParam != "" {
			redirectURL += "&mods=" + modsParam
			if e.Request.FormValue("mod_match") == "all" {
				redirectURL += "&mod_match=all"
			}
		}
		return e.Redirect(http.StatusTemporaryRedirect, LangRedirectURL(e, redirectURL))
	}
}

func allTagsFromStore(appStore *store.Store) []models.SchematicTag {
	tags, err := appStore.Tags.List(context.Background())
	if err != nil {
		return nil
	}
	result := make([]models.SchematicTag, len(tags))
	for i, t := range tags {
		result[i] = models.SchematicTag{
			ID:   t.ID,
			Key:  t.Key,
			Name: t.Name,
		}
	}
	return result
}

func allTagsWithCountFromStore(appStore *store.Store, service *cache.Service) []models.SchematicTagWithCount {
	tagsWithCount, found := service.GetTagWithCount(cache.AllTagsWithCountKey)
	if found {
		return tagsWithCount
	}
	tags, err := appStore.Tags.ListWithCount(context.Background())
	if err != nil {
		return nil
	}
	result := make([]models.SchematicTagWithCount, len(tags))
	for i, t := range tags {
		result[i] = models.SchematicTagWithCount{
			ID:    t.ID,
			Key:   t.Key,
			Name:  t.Name,
			Count: t.Count,
		}
	}
	service.SetWithTTL(cache.AllTagsWithCountKey, result, 6*time.Hour)
	return result
}

// allModOptionsFromStore returns mod options with display names for the search filter.
func allModOptionsFromStore(appStore *store.Store, cacheService *cache.Service) []ModOption {
	const cacheKey = "search_mod_options"
	if val, found := cacheService.Get(cacheKey); found {
		if opts, ok := val.([]ModOption); ok {
			return opts
		}
	}
	ctx := context.Background()
	modCounts, err := appStore.Schematics.ListModCounts(ctx)
	if err != nil {
		return nil
	}
	opts := make([]ModOption, 0, len(modCounts))
	for _, mc := range modCounts {
		name := strings.TrimSpace(mc.ModName)
		if name == "" {
			continue
		}
		displayName := name
		meta, err := appStore.ModMetadata.GetByNamespace(ctx, name)
		if err == nil && meta != nil && meta.DisplayName != "" {
			displayName = meta.DisplayName
		}
		opts = append(opts, ModOption{Namespace: name, DisplayName: displayName, Count: mc.Count})
	}
	cacheService.SetWithTTL(cacheKey, opts, 6*time.Hour)
	return opts
}
