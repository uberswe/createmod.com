package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/store"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"strconv"
	"strings"
	"time"
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
}

func SearchHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		term := strings.ReplaceAll(slugTerm, "-", " ")
		app.Logger().Debug("search", "term", term, "searchService", searchService)
		ids := searchService.Search(term, order, rating, category, searchTags, mcVersion, createVersion)
		app.Logger().Debug("found ids", "ids", ids)

		interfaceIds := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			interfaceIds = append(interfaceIds, id)
		}

		var res []*core.Record
		err := app.RecordQuery("schematics").
			Select("schematics.*").
			From("schematics").
			Where(dbx.NewExp("(deleted = '' OR deleted IS NULL) AND moderated = true AND (scheduled_at IS NULL OR scheduled_at <= DATETIME('now'))")).
			Where(dbx.In("id", interfaceIds...)).
			All(&res)

		if err != nil {
			return err
		}
		sortedModels := make([]*core.Record, 0)
		for id := range ids {
			for i := range res {
				if ids[id] == res[i].Id {
					sortedModels = append(sortedModels, res[i])
				}
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
		pageSize := 24
		total := len(sortedModels)
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
		pageSlice := sortedModels
		if total > 0 {
			pageSlice = sortedModels[startIdx:endIdx]
		}

		schematicModels := MapResultsToSchematic(app, pageSlice, cacheService)

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
			Tags:              allTags(app),
			TagsWithCount:     allTagsWithCount(app, cacheService),
			SelectedTags:      selectedTags,
			MinecraftVersions: allMinecraftVersions(app),
			CreateVersions:    allCreatemodVersions(app),
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
		}
		d.Populate(e)
		// Dynamic title based on search context
		if term != "" && page > 1 {
			d.Title = fmt.Sprintf("%s "+i18n.T(d.Language, "Schematics - Page")+" %d", term, page)
		} else if term != "" {
			d.Title = fmt.Sprintf("%s "+i18n.T(d.Language, "Schematics"), term)
		} else {
			d.Title = i18n.T(d.Language, "Search")
		}
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
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
			d.Thumbnail = fmt.Sprintf("https://createmod.com/api/files/schematics/%s/%s", d.Schematics[0].ID, d.Schematics[0].FeaturedImage)
		}

		html, err := registry.LoadFiles(searchTemplates...).Render(d)
		if err != nil {
			return err
		}
		// Update search count
		err = searchCount(app, term, slugTerm, int32(d.SearchResultCount))
		if err != nil {
			return err
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

func searchCount(app *pocketbase.PocketBase, term string, termSlug string, searchResults int32) error {
	term = strings.ToLower(strings.TrimSpace(term))
	// Skip empty or invalid searches
	if term == "" {
		return nil
	}
	records, err := app.FindRecordsByFilter("searches", "term = {:term}", "+term", 1, 0, dbx.Params{"term": term})
	if err != nil {
		return err
	}
	searchesCollection, err := app.FindCollectionByNameOrId("searches")
	if err != nil {
		return err
	}
	if len(records) == 0 {
		record := core.NewRecord(searchesCollection)
		record.Set("term", term)
		record.Set("slug", termSlug)
		record.Set("searches", 1)
		record.Set("results", searchResults)
		return app.Save(record)
	}
	record := records[0]
	record.Set("searches", record.GetInt("searches")+1)
	record.Set("results", searchResults)
	return app.Save(record)
}

func SearchPostHandler(app *pocketbase.PocketBase, service *cache.Service, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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
			return apis.NewBadRequestError("Failed to read request data", err)
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

func allTags(app *pocketbase.PocketBase) []models.SchematicTag {
	tagsCollection, err := app.FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(tagsCollection.Id, "1=1", "+name", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToTags(records)
}

type schematicTags struct {
	Tags string
}

func allTagsWithCount(app *pocketbase.PocketBase, service *cache.Service) []models.SchematicTagWithCount {
	tagsWithCount, found := service.GetTagWithCount(cache.AllTagsWithCountKey)
	if found {
		return tagsWithCount
	}
	tags := allTags(app)
	var schematics []schematicTags
	err := app.DB().
		Select("schematics.tags").
		From("schematics").
		All(&schematics)
	if err != nil {
		app.Logger().Debug("could not fetch tags with count", "error", err.Error())
		return nil
	}
	for i := range tags {
		tagsWithCount = append(tagsWithCount, models.SchematicTagWithCount{
			ID:    tags[i].ID,
			Key:   tags[i].Key,
			Name:  tags[i].Name,
			Count: 0,
		})
		for x := range schematics {
			if strings.Contains(schematics[x].Tags, tagsWithCount[i].ID) {
				tagsWithCount[i].Count++
			}
		}
	}
	service.SetTagWithCount(cache.AllTagsWithCountKey, tagsWithCount)
	return tagsWithCount
}
