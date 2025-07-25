package search

import (
	"createmod/internal/models"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/pocketbase/pocketbase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	bestMatchOrder     = 1
	newestOrder        = 2
	oldestOrder        = 3
	highestRatingOrder = 4
	lowestRatingOrder  = 5
	mostViewedOrder    = 6
	leastViewedOrder   = 7
	regex              = `<.*?>`
)

type Service struct {
	index      []schematicIndex
	bleveIndex bleve.Index
	app        *pocketbase.PocketBase
}

type schematicIndex struct {
	ID               string
	Title            string
	Description      string
	Created          time.Time
	Tags             []string
	Categories       []string
	Views            int64
	Rating           float64
	Author           string
	MinecraftVersion string
	CreateVersion    string
}

type bleveIndex struct {
	Title       string
	Description string
	Tags        []string
	Categories  []string
	Author      string
}

func New(schematics []models.Schematic, app *pocketbase.PocketBase) *Service {
	s := Service{}
	s.app = app
	mapping := bleve.NewIndexMapping()

	schematicMapping := bleve.NewDocumentMapping()

	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Name = "title"
	schematicMapping.AddFieldMappingsAt("title", titleFieldMapping)
	descriptionFieldMapping := bleve.NewTextFieldMapping()
	descriptionFieldMapping.Name = "description"
	schematicMapping.AddFieldMappingsAt("description", descriptionFieldMapping)
	tagsFieldMapping := bleve.NewTextFieldMapping()
	schematicMapping.AddFieldMappingsAt("tags", tagsFieldMapping)
	categoriesFieldMapping := bleve.NewTextFieldMapping()
	schematicMapping.AddFieldMappingsAt("categories", categoriesFieldMapping)
	authorFieldMapping := bleve.NewTextFieldMapping()
	schematicMapping.AddFieldMappingsAt("author", authorFieldMapping)

	mapping.AddDocumentMapping("schematic", schematicMapping)
	var err error
	s.bleveIndex, err = bleve.NewMemOnly(mapping)
	if err != nil {
		panic(err)
	}
	s.BuildIndex(schematics)
	return &s
}

// Search takes a term and returns schematic ids in the specified order
func (s *Service) Search(term string, order int, rating int, category string, tag string, minecraftVersion string, createVersion string) []string {
	// If search hasn't had time to initialize, usually after a reboot
	s.app.Logger().Debug("starting search service - check if initialized")
	if s == nil || s.index == nil {
		s.app.Logger().Debug("search service is down", "search", s)
		return nil
	}

	// Ratings
	result := make([]schematicIndex, len(s.index))
	copy(result, s.index)
	s.app.Logger().Debug("searching schematics", "index count", len(s.index), "result", len(result))
	if rating > 0 {
		ratingFloat := float64(rating)
		ratingResult := make([]schematicIndex, 0)
		for i := range result {
			if result[i].Rating >= ratingFloat {
				ratingResult = append(ratingResult, result[i])
			}
		}
		result = ratingResult
	}

	s.app.Logger().Debug("filtered by rating", "count", len(result), "rating", rating)
	// Category
	if category != "all" {
		categoryResult := make([]schematicIndex, 0)
		for i := range result {
			cat := strings.ReplaceAll(category, "-", " ")
			caser := cases.Title(language.English)
			if slices.Contains(result[i].Categories, caser.String(cat)) {
				categoryResult = append(categoryResult, result[i])
			}
		}
		result = categoryResult
	}
	s.app.Logger().Debug("filtered by category", "count", len(result), "category", category)
	// Tag
	if tag != "all" {
		tagResult := make([]schematicIndex, 0)
		for i := range result {
			tag = strings.ReplaceAll(tag, "-", " ")
			caser := cases.Title(language.English)
			if slices.Contains(result[i].Tags, caser.String(tag)) {
				tagResult = append(tagResult, result[i])
			}
		}
		result = tagResult
	}
	s.app.Logger().Debug("filtered by tag", "count", len(result), "tag", tag)
	// Create Mod Version
	if createVersion != "all" {
		cvResult := make([]schematicIndex, 0)
		for i := range result {
			if result[i].CreateVersion == createVersion {
				cvResult = append(cvResult, result[i])
			}
		}
		result = cvResult
	}
	s.app.Logger().Debug("filtered by create mod version", "count", len(result), "createVersion", createVersion)
	// Minecraft version
	if minecraftVersion != "all" {
		mcvResult := make([]schematicIndex, 0)
		for i := range result {
			if result[i].MinecraftVersion == minecraftVersion {
				mcvResult = append(mcvResult, result[i])
			}
		}
		result = mcvResult
	}
	s.app.Logger().Debug("filtered by minecraft version", "count", len(result), "minecraftVersion", minecraftVersion)
	// Bleve
	if strings.TrimSpace(term) != "" {
		newResult := make([]schematicIndex, 0)
		queryFormat := fmt.Sprintf("Title:%s^5 Description:%s Tags:%s^2 Categories:%s^2 Author:%s^3", term, term, term, term, term)
		if strings.Contains(term, " ") {
			queryFormat = term
		}
		// https://blevesearch.com/docs/Query-String-Query/
		query := bleve.NewQueryStringQuery(queryFormat)
		q, err := query.Parse()
		fields, fieldsError := s.bleveIndex.Fields()
		s.app.Logger().Debug("searching schematics", "term", term, "query", query.Query, "error", err, "fullQuery", q, "fieldsError", fieldsError, "fields", fields)
		searchRequest := bleve.NewSearchRequest(query)
		searchRequest.Size = 5000
		searchResult, err := s.bleveIndex.Search(searchRequest)
		if err != nil {
			s.app.Logger().Error("error for bleve search query", "error", err.Error())
		}
		if searchResult != nil {
			count, err := s.bleveIndex.DocCount()
			s.app.Logger().Debug("bleve search results", "total", searchResult.Total, "hits", len(searchResult.Hits), "stats", s.bleveIndex.StatsMap(), "index", count, "error", err)
			for _, si := range searchResult.Hits {
				for i := range result {
					if result[i].ID == si.ID {
						newResult = append(newResult, result[i])
					}
				}
			}
		}
		result = newResult
		s.app.Logger().Debug("filtered by bleve", "count", len(result))
	}
	// Order
	slices.SortFunc(result, func(a, b schematicIndex) int {
		switch order {
		case bestMatchOrder:
			// Handled by bleve
			return 0
		case newestOrder:
			return newestSort(a, b)
		case oldestOrder:
			return -newestSort(a, b)
		case highestRatingOrder:
			return highestRatingSort(a, b)
		case lowestRatingOrder:
			return -highestRatingSort(a, b)
		case mostViewedOrder:
			return mostViewedSort(a, b)
		case leastViewedOrder:
			return -mostViewedSort(a, b)
		default:
			return 0
		}
	})
	s.app.Logger().Debug("sorted", "count", len(result))

	ids := make([]string, len(result))
	for i := range result {
		ids[i] = result[i].ID
	}
	s.app.Logger().Debug("returning ids", "count", len(ids))
	return ids
}

func mostViewedSort(a schematicIndex, b schematicIndex) int {
	if a.Views >= b.Views {
		return -1
	}
	return 1
}

func highestRatingSort(a schematicIndex, b schematicIndex) int {
	if a.Rating >= b.Rating {
		return -1
	}
	return 1
}

func newestSort(a schematicIndex, b schematicIndex) int {
	if a.Created.Before(b.Created) {
		return 1
	}
	return -1
}

// BuildIndex takes a set of schematics and prepares a search index
func (s *Service) BuildIndex(schematics []models.Schematic) {
	batch := s.bleveIndex.NewBatch()
	index := make([]schematicIndex, len(schematics))
	for i := range schematics {
		index[i] = schematicIndex{
			ID:          schematics[i].ID,
			Title:       stripHtmlRegex(schematics[i].Title),
			Description: stripHtmlRegex(schematics[i].Content),
			Created:     schematics[i].Created,
			Views:       int64(schematics[i].Views),
			Author:      schematics[i].Author.Username,
		}
		if parsedFloat, err := strconv.ParseFloat(schematics[i].Rating, 64); err == nil {
			index[i].Rating = parsedFloat
		}
		for _, c := range schematics[i].Categories {
			index[i].Categories = append(index[i].Categories, c.Name)
		}
		for _, t := range schematics[i].Tags {
			index[i].Tags = append(index[i].Tags, t.Name)
		}
		err := batch.Index(index[i].ID, bleveIndex{
			Title:       index[i].Title,
			Description: index[i].Description,
			Tags:        index[i].Tags,
			Categories:  index[i].Categories,
			Author:      index[i].Author,
		})

		index[i].MinecraftVersion = schematics[i].MinecraftVersion
		index[i].CreateVersion = schematics[i].CreatemodVersion

		if err != nil {
			s.app.Logger().Error("bleve add index", "error", err.Error())
		}
	}
	err := s.bleveIndex.Batch(batch)
	if err != nil {
		s.app.Logger().Error("bleve search batching", "error", err.Error())
		return
	}
	ids := make([]string, len(index))
	for i, in := range index {
		ids[i] = in.ID
	}
	s.index = index
}

func stripHtmlRegex(s string) string {
	r := regexp.MustCompile(regex)
	return r.ReplaceAllString(s, " ")
}
