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
		var schematics []models.DatabaseSchematic
		app.Logger().Debug("search", "term", term)
		ids := searchService.Search(term, order, rating, category, tag)
		app.Logger().Debug("found ids", "ids", ids)
		interfaceIds := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			interfaceIds = append(interfaceIds, id)
		}

		err := app.Dao().DB().
			Select("schematics.*").
			From("schematics").
			Where(dbx.In("id", interfaceIds...)).
			All(&schematics)

		if err != nil {
			return err
		}
		schematicModels := models.DatabaseSchematicsToSchematics(schematics)
		sortedModels := make([]models.Schematic, 0)
		for id := range ids {
			for i := range schematicModels {
				if ids[id] == schematicModels[i].ID {
					sortedModels = append(sortedModels, schematicModels[i])
				}
			}
		}

		end := time.Now()
		duration := end.Sub(start)
		d := SearchData{
			Schematics:        sortedModels,
			Tags:              allTags(app),
			SearchSpeed:       fmt.Sprintf("%.6f", duration.Seconds()),
			SearchResultCount: len(ids),
			Term:              term,
			Sort:              order,
			Rating:            rating,
			Category:          category,
		}
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
