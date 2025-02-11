package pages

import (
	"createmod/internal/models"
	"createmod/internal/search"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	pbmodels "github.com/pocketbase/pocketbase/models"
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

func SearchHandler(app *pocketbase.PocketBase, searchService *search.Service) func(c echo.Context) error {
	return func(c echo.Context) error {
		start := time.Now()
		slugTerm := c.PathParam("term")
		order := 1
		if c.QueryParam("sort") != "" {
			atoi, err := strconv.Atoi(c.QueryParam("sort"))
			if err != nil {
				return err
			}
			order = atoi
		}
		rating := -1
		if c.QueryParam("rating") != "" {
			atoi, err := strconv.Atoi(c.QueryParam("rating"))
			if err != nil {
				return err
			}
			rating = atoi
		}
		category := "all"
		if c.QueryParam("category") != "" {
			category = c.QueryParam("category")
		}
		tag := "all"
		if c.QueryParam("tag") != "" {
			tag = c.QueryParam("tag")
		}

		term := strings.ReplaceAll(slugTerm, "-", " ")
		app.Logger().Debug("search", "term", term, "searchService", searchService)
		ids := searchService.Search(term, order, rating, category, tag)
		app.Logger().Debug("found ids", "ids", ids)

		interfaceIds := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			interfaceIds = append(interfaceIds, id)
		}

		var res []*pbmodels.Record
		err := app.Dao().RecordQuery("schematics").
			Select("schematics.*").
			From("schematics").
			Where(dbx.In("id", interfaceIds...)).
			All(&res)

		if err != nil {
			return err
		}
		sortedModels := make([]*pbmodels.Record, 0)
		for id := range ids {
			for i := range res {
				if ids[id] == res[i].GetId() {
					sortedModels = append(sortedModels, res[i])
				}
			}
		}
		limit := 100
		if len(sortedModels) > limit {
			sortedModels = sortedModels[:limit]
		}

		schematicModels := MapResultsToSchematic(app, sortedModels)

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
		d.Populate(c)
		d.Title = "Search"
		d.Categories = allCategories(app)

		err = c.Render(http.StatusOK, searchTemplate, d)
		if err != nil {
			return err
		}
		return nil
	}
}

func SearchPostHandler(app *pocketbase.PocketBase) func(c echo.Context) error {
	return func(c echo.Context) error {
		data := struct {
			Term     string `json:"term" form:"advanced-search-term"`
			Sort     string `json:"sort" form:"advanced-search-sort"`
			Rating   string `json:"rating" form:"advanced-search-ranking"`
			Category string `json:"category" form:"advanced-search-category"`
			Tag      string `json:"tag" form:"advanced-search-tag"`
		}{}
		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Failed to read request data", err)
		}
		term := slug.Make(data.Term)
		return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/search/%s?sort=%s&rating=%s&category=%s&tag=%s", term, data.Sort, data.Rating, data.Category, data.Tag))
	}
}

func allTags(app *pocketbase.PocketBase) []models.SchematicTag {
	tagsCollection, err := app.Dao().FindCollectionByNameOrId("schematic_tags")
	if err != nil {
		return nil
	}
	records, err := app.Dao().FindRecordsByFilter(tagsCollection.Id, "1=1", "+name", -1, 0)
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
	err := app.Dao().DB().
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
