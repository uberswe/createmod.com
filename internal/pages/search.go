package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/metrics"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/store"
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
	"./template/include/schematic_card_medium.html",
	"./template/include/search_filters.html",
	"./template/include/search_pagination.html",
}, commonTemplates...)

type SearchData struct {
	DefaultData
	Schematics        []models.Schematic
	Tags              []models.SchematicTag
	TagsWithCount     []models.SchematicTagWithCount
	SelectedTags      []string
	MinecraftVersions []models.MinecraftVersion
	CreateVersions    []models.CreatemodVersion
	SearchSpeed       string
	SearchResultCount int // total results count
	TotalResults      int
	TotalPages        int
	PageNumbers       []int // sliding window page numbers; -1 = ellipsis
	Term              string
	TermSlug          string
	Sort              int
	DisplaySort       int // sort value shown in the UI (always BestMatch when no explicit sort)
	Rating            int
	Category          string
	Tag               string // backward compat: first selected tag
	MinecraftVersion  string
	CreateVersion     string
	Page              int
	PageSize          int
	HasPrev           bool
	HasNext           bool
	PrevURL           string
	NextURL           string
	ViewMode          string
	HidePaid          bool
}

func SearchHandler(searchEngine search.SearchEngine, cacheService *cache.Service, registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
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

		hidePaid := e.Request.URL.Query().Get("hidepaid") == "1"

		term := strings.ReplaceAll(slugTerm, "-", " ")

		sq := search.SearchQuery{
			Term:             term,
			Order:            order,
			Rating:           rating,
			Category:         category,
			Tags:             searchTags,
			MinecraftVersion: mcVersion,
			CreateVersion:    createVersion,
			HidePaid:         hidePaid,
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
				if s.Deleted != nil || !s.Moderated {
					continue
				}
				orderedSchematics = append(orderedSchematics, s)
			}
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
		pageSize := 18
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
		if hidePaid {
			queryParts = append(queryParts, "hidepaid=1")
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

		// SEO canonical & prev/next
		canonicalURL := fmt.Sprintf("https://createmod.com%s", buildPageURL(page))
		seoNoIndex := page > 20

		viewMode := e.Request.URL.Query().Get("view")
		if viewMode != "list" {
			viewMode = "grid"
		}

		totalPages := 0
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
		}

		d := SearchData{
			Schematics:        schematicModels,
			Tags:              allTagsFromStore(appStore),
			TagsWithCount:     allTagsWithCountFromStore(appStore, cacheService),
			SelectedTags:      selectedTags,
			MinecraftVersions: allMinecraftVersionsFromStore(appStore),
			CreateVersions:    allCreatemodVersionsFromStore(appStore),
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
			MinecraftVersion:  mcVersion,
			CreateVersion:     createVersion,
			Page:              page,
			PageSize:          pageSize,
			HasPrev:           prevURL != "",
			HasNext:           nextURL != "",
			PrevURL:           prevURL,
			NextURL:           nextURL,
			ViewMode:          viewMode,
			HidePaid:          hidePaid,
		}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Search"))
		// Dynamic title based on search context
		if term != "" && page > 1 {
			d.Title = fmt.Sprintf("%s "+i18n.T(d.Language, "Schematics - Page")+" %d", term, page)
		} else if term != "" {
			d.Title = fmt.Sprintf("%s "+i18n.T(d.Language, "Schematics"), term)
		} else {
			d.Title = i18n.T(d.Language, "Search")
		}
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Description = fmt.Sprintf(i18n.T(d.Language, "page.search.description"), d.Term)
		d.Slug = fmt.Sprintf("/search/%s", slugTerm)
		d.CanonicalURL = canonicalURL
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
		// Update search count via store
		_ = appStore.SearchTracking.RecordSearch(ctx, term, d.SearchResultCount, "", "")
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
		term := slug.Make(data.Term)
		return e.Redirect(http.StatusTemporaryRedirect, LangRedirectURL(e, fmt.Sprintf("/search/%s?sort=%s&rating=%s&category=%s&tag=%s&mcv=%s&cv=%s", term, data.Sort, data.Rating, data.Category, tagParam, data.MinecraftVersion, data.CreateVersion)))
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
