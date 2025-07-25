package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/search"
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

var searchTemplates = []string{
	"./template/dist/search.html",
	"./template/dist/include/schematic_card.html",
}

type SearchData struct {
	DefaultData
	Schematics        []models.Schematic
	Tags              []models.SchematicTag
	MinecraftVersions []models.MinecraftVersion
	CreateVersions    []models.CreatemodVersion
	SearchSpeed       string
	SearchResultCount int
	Term              string
	Sort              int
	Rating            int
	Category          string
	Tag               string
	MinecraftVersion  string
	CreateVersion     string
}

func SearchHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		start := time.Now()
		slugTerm := e.Request.PathValue("term")
		order := 6
		if e.Request.URL.Query().Get("sort") != "" {
			atoi, err := strconv.Atoi(e.Request.URL.Query().Get("sort"))
			if err != nil {
				return err
			}
			order = atoi
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
		tag := "all"
		if e.Request.URL.Query().Get("tag") != "" {
			tag = e.Request.URL.Query().Get("tag")
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
		ids := searchService.Search(term, order, rating, category, tag, mcVersion, createVersion)
		app.Logger().Debug("found ids", "ids", ids)

		interfaceIds := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			interfaceIds = append(interfaceIds, id)
		}

		var res []*core.Record
		err := app.RecordQuery("schematics").
			Select("schematics.*").
			From("schematics").
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
		limit := 100
		if len(sortedModels) > limit {
			sortedModels = sortedModels[:limit]
		}

		schematicModels := MapResultsToSchematic(app, sortedModels, cacheService)

		end := time.Now()
		duration := end.Sub(start)
		d := SearchData{
			Schematics:        schematicModels,
			Tags:              allTags(app),
			MinecraftVersions: allMinecraftVersions(app),
			CreateVersions:    allCreatemodVersions(app),
			SearchSpeed:       fmt.Sprintf("%.6f", duration.Seconds()),
			SearchResultCount: len(sortedModels),
			Term:              term,
			Sort:              order,
			Rating:            rating,
			Category:          category,
			MinecraftVersion:  mcVersion,
			CreateVersion:     createVersion,
		}
		d.Populate(e)
		d.Title = "Search"
		d.Categories = allCategories(app, cacheService)
		d.Description = fmt.Sprintf("Search results for %s Create Mod Schematics.", d.Term)
		d.Slug = fmt.Sprintf("/search/%s", slugTerm)
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

func SearchPostHandler(app *pocketbase.PocketBase, service *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := struct {
			Term             string `json:"term" form:"advanced-search-term"`
			Sort             string `json:"sort" form:"advanced-search-sort"`
			Rating           string `json:"rating" form:"advanced-search-ranking"`
			Category         string `json:"category" form:"advanced-search-category"`
			Tag              string `json:"tag" form:"advanced-search-tag"`
			MinecraftVersion string `json:"minecraft-version" form:"advanced-search-minecraft-version"`
			CreateVersion    string `json:"create-version" form:"advanced-search-create-version"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return apis.NewBadRequestError("Failed to read request data", err)
		}
		term := slug.Make(data.Term)
		return e.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/search/%s?sort=%s&rating=%s&category=%s&tag=%s&mcv=%s&cv=%s", term, data.Sort, data.Rating, data.Category, data.Tag, data.MinecraftVersion, data.CreateVersion))
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
