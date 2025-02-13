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

const searchTemplate = "search.html"

type SearchData struct {
	DefaultData
	Schematics        []models.Schematic
	Tags              []models.SchematicTag
	SearchSpeed       string
	SearchResultCount int
	Term              string
	Sort              int
	Rating            int
	Category          string
	Tag               string
}

func SearchHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		start := time.Now()
		slugTerm := e.Request.PathValue("term")
		order := 1
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

		term := strings.ReplaceAll(slugTerm, "-", " ")
		app.Logger().Debug("search", "term", term, "searchService", searchService)
		ids := searchService.Search(term, order, rating, category, tag)
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
			SearchSpeed:       fmt.Sprintf("%.6f", duration.Seconds()),
			SearchResultCount: len(sortedModels),
			Term:              term,
			Sort:              order,
			Rating:            rating,
			Category:          category,
		}
		d.Populate(e)
		d.Title = "Search"
		d.Categories = allCategories(app)

		html, err := registry.LoadFiles(searchTemplate).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func SearchPostHandler(app *pocketbase.PocketBase, service *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		data := struct {
			Term     string `json:"term" form:"advanced-search-term"`
			Sort     string `json:"sort" form:"advanced-search-sort"`
			Rating   string `json:"rating" form:"advanced-search-ranking"`
			Category string `json:"category" form:"advanced-search-category"`
			Tag      string `json:"tag" form:"advanced-search-tag"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return apis.NewBadRequestError("Failed to read request data", err)
		}
		term := slug.Make(data.Term)
		return e.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/search/%s?sort=%s&rating=%s&category=%s&tag=%s", term, data.Sort, data.Rating, data.Category, data.Tag))
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

func allTagsWithCount(app *pocketbase.PocketBase) []models.SchematicTagWithCount {
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
	tagsWithCount := make([]models.SchematicTagWithCount, len(tags))
	for i := range tags {
		tagsWithCount[i] = models.SchematicTagWithCount{
			ID:    tags[i].ID,
			Key:   tags[i].Key,
			Name:  tags[i].Name,
			Count: 0,
		}
		for x := range schematics {
			if strings.Contains(schematics[x].Tags, tagsWithCount[i].ID) {
				tagsWithCount[i].Count++
			}
		}
	}
	return tagsWithCount
}
