package search

import (
	"createmod/internal/models"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
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
	BestMatchOrder     = 1
	NewestOrder        = 2
	OldestOrder        = 3
	HighestRatingOrder = 4
	LowestRatingOrder  = 5
	MostViewedOrder    = 6
	LeastViewedOrder   = 7
	TrendingOrder      = 8
	regex              = `<.*?>`
)

type Service struct {
	index          []schematicIndex
	bleveIndex     bleve.Index
	app            *pocketbase.PocketBase
	trendingScores map[string]float64
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
	Title         string
	Description   string
	AIDescription string
	Tags          []string
	Categories    []string
	Author        string
}

// SetTrendingScores sets the trending scores map used for trending sort order.
func (s *Service) SetTrendingScores(scores map[string]float64) {
	s.trendingScores = scores
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
	aiDescFieldMapping := bleve.NewTextFieldMapping()
	aiDescFieldMapping.Name = "aidescription"
	schematicMapping.AddFieldMappingsAt("aidescription", aiDescFieldMapping)
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

// Search takes a term and returns schematic ids in the specified order.
// tags is a list of tags to filter by (AND logic — result must match ALL selected tags).
// Pass nil or empty slice for no tag filtering.
func (s *Service) Search(term string, order int, rating int, category string, tags []string, minecraftVersion string, createVersion string) []string {
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
	// Tags (AND logic: result must match ALL selected tags)
	if len(tags) > 0 && !(len(tags) == 1 && tags[0] == "all") {
		caser := cases.Title(language.English)
		tagResult := make([]schematicIndex, 0)
		for i := range result {
			matchAll := true
			for _, tag := range tags {
				normalized := caser.String(strings.ReplaceAll(tag, "-", " "))
				if !slices.Contains(result[i].Tags, normalized) {
					matchAll = false
					break
				}
			}
			if matchAll {
				tagResult = append(tagResult, result[i])
			}
		}
		result = tagResult
	}
	s.app.Logger().Debug("filtered by tags", "count", len(result), "tags", tags)
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

		// Build a disjunction of: AND-match (all words must appear) + exact phrase boost
		words := strings.Fields(term)
		var searchQuery query.Query
		if len(words) == 1 {
			// Single word: use query string with field boosts
			searchQuery = bleve.NewQueryStringQuery(term)
		} else {
			// Multi-word: conjunction (AND) of each word across any field
			conjuncts := make([]query.Query, 0, len(words))
			for _, w := range words {
				conjuncts = append(conjuncts, bleve.NewMatchQuery(w))
			}
			andQuery := bleve.NewConjunctionQuery(conjuncts...)

			// Exact phrase boost (10x)
			phraseQuery := bleve.NewMatchPhraseQuery(term)
			phraseQuery.SetBoost(10.0)

			// Combine: results matching AND or phrase, phrase-matched results score higher
			searchQuery = bleve.NewDisjunctionQuery(andQuery, phraseQuery)
		}

		searchRequest := bleve.NewSearchRequest(searchQuery)
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
		case BestMatchOrder:
			// Handled by bleve
			return 0
		case NewestOrder:
			return newestSort(a, b)
		case OldestOrder:
			return -newestSort(a, b)
		case HighestRatingOrder:
			return highestRatingSort(a, b)
		case LowestRatingOrder:
			return -highestRatingSort(a, b)
		case MostViewedOrder:
			return mostViewedSort(a, b)
		case LeastViewedOrder:
			return -mostViewedSort(a, b)
		case TrendingOrder:
			return trendingSort(s.trendingScores, a, b)
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

func trendingSort(scores map[string]float64, a schematicIndex, b schematicIndex) int {
	sa := scores[a.ID]
	sb := scores[b.ID]
	if sa > sb {
		return -1
	}
	if sa < sb {
		return 1
	}
	return 0
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
			Title:         index[i].Title,
			Description:   index[i].Description,
			AIDescription: stripHtmlRegex(schematics[i].AIDescription),
			Tags:          index[i].Tags,
			Categories:    index[i].Categories,
			Author:        index[i].Author,
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

// Suggestion represents an autocomplete suggestion result.
type Suggestion struct {
	Text string `json:"text"`
	Type string `json:"type"` // "schematic", "tag", "category"
	URL  string `json:"url"`
}

// Suggest returns autocomplete suggestions matching the given query prefix.
// It searches titles, tags, and categories from the in-memory index.
func (s *Service) Suggest(q string, limit int) []Suggestion {
	if s == nil || s.index == nil || len(q) < 2 {
		return nil
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}

	results := make([]Suggestion, 0, limit)
	seen := make(map[string]bool)

	// Title matches (prefix then contains)
	for _, idx := range s.index {
		if len(results) >= limit {
			break
		}
		titleLower := strings.ToLower(idx.Title)
		if strings.HasPrefix(titleLower, q) || strings.Contains(titleLower, q) {
			key := "s:" + idx.ID
			if !seen[key] {
				seen[key] = true
				results = append(results, Suggestion{
					Text: idx.Title,
					Type: "schematic",
					URL:  "/schematics/" + idx.ID,
				})
			}
		}
	}

	// Tag name matches
	tagSeen := make(map[string]bool)
	for _, idx := range s.index {
		if len(results) >= limit {
			break
		}
		for _, tag := range idx.Tags {
			tagLower := strings.ToLower(tag)
			if tagSeen[tagLower] {
				continue
			}
			if strings.HasPrefix(tagLower, q) || strings.Contains(tagLower, q) {
				tagSeen[tagLower] = true
				caser := cases.Title(language.English)
				tagKey := strings.ReplaceAll(strings.ToLower(tag), " ", "-")
				results = append(results, Suggestion{
					Text: caser.String(tag),
					Type: "tag",
					URL:  "/search/?tag=" + tagKey,
				})
				if len(results) >= limit {
					break
				}
			}
		}
	}

	// Category name matches
	catSeen := make(map[string]bool)
	for _, idx := range s.index {
		if len(results) >= limit {
			break
		}
		for _, cat := range idx.Categories {
			catLower := strings.ToLower(cat)
			if catSeen[catLower] {
				continue
			}
			if strings.HasPrefix(catLower, q) || strings.Contains(catLower, q) {
				catSeen[catLower] = true
				catKey := strings.ReplaceAll(strings.ToLower(cat), " ", "-")
				results = append(results, Suggestion{
					Text: cat,
					Type: "category",
					URL:  "/search/?category=" + catKey,
				})
				if len(results) >= limit {
					break
				}
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func stripHtmlRegex(s string) string {
	r := regexp.MustCompile(regex)
	return r.ReplaceAllString(s, " ")
}
