package search

import (
	"bytes"
	"compress/gzip"
	"context"
	"createmod/internal/models"
	"createmod/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	// cacheKey is the storage path for the serialized index cache.
	cacheKey = "_internal/search_index_cache.json.gz"
)

type Service struct {
	index          []schematicIndex
	bleveIndex     bleve.Index
	storage        *storage.Service
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
	Paid             bool
}

type bleveIndex struct {
	Title         string
	Description   string
	AIDescription string
	Tags          []string
	Categories    []string
	Author        string
}

// indexCacheEntry holds both filter-index and Bleve-index data for one schematic.
type indexCacheEntry struct {
	SI schematicIndex `json:"si"`
	BI bleveIndex     `json:"bi"`
}

// Ready returns true when the search index has been populated and is ready
// to serve queries. Used by the readiness probe to delay traffic until the
// pod can actually produce search results.
func (s *Service) Ready() bool {
	return s != nil && s.index != nil && len(s.index) > 0
}

// SetTrendingScores sets the trending scores map used for trending sort order.
func (s *Service) SetTrendingScores(scores map[string]float64) {
	s.trendingScores = scores
}

// newBleveIndex creates a fresh in-memory Bleve index with the schematic mapping.
func newBleveIndex() bleve.Index {
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

	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		panic(err)
	}
	return idx
}

// New creates a search Service with an in-memory Bleve index.
// On startup it attempts to load a cached index snapshot from storage (S3)
// so the server can serve search requests immediately while a background
// rebuild picks up recent changes.
func New(storageSvc *storage.Service) *Service {
	s := Service{storage: storageSvc}
	s.bleveIndex = newBleveIndex()

	// Try to warm from storage cache.
	if err := s.loadCacheFromStorage(); err != nil {
		slog.Info("search: no usable cache in storage, starting empty", "error", err)
	} else {
		slog.Info("search: loaded index cache from storage", "docs", len(s.index))
	}

	return &s
}

// NewEmpty creates a search service without loading the S3 cache, so the
// caller is not blocked by network I/O. The index starts empty and is
// populated by the River SearchIndexWorker which runs on start.
func NewEmpty(storageSvc *storage.Service) *Service {
	s := Service{storage: storageSvc}
	s.bleveIndex = newBleveIndex()
	return &s
}

// WarmFromStorage attempts to load a cached index snapshot from S3.
// This is safe to call concurrently; BuildIndex will overwrite the
// cached data when a full rebuild completes.
func (s *Service) WarmFromStorage() {
	if err := s.loadCacheFromStorage(); err != nil {
		slog.Info("search: no usable cache in storage, starting empty", "error", err)
	} else {
		slog.Info("search: loaded index cache from storage", "docs", len(s.index))
	}
}

// Search takes a term and returns schematic ids in the specified order.
// tags is a list of tags to filter by (AND logic — result must match ALL selected tags).
// Pass nil or empty slice for no tag filtering.
func (s *Service) Search(term string, order int, rating int, category string, tags []string, minecraftVersion string, createVersion string, hidePaid bool) []string {
	// If search hasn't had time to initialize, usually after a reboot
	slog.Debug("starting search service - check if initialized")
	if s == nil || s.index == nil {
		slog.Debug("search service is down")
		return nil
	}

	// Ratings
	result := make([]schematicIndex, len(s.index))
	copy(result, s.index)
	slog.Debug("searching schematics", "index count", len(s.index), "result", len(result))
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

	slog.Debug("filtered by rating", "count", len(result), "rating", rating)
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
	slog.Debug("filtered by category", "count", len(result), "category", category)
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
	slog.Debug("filtered by tags", "count", len(result), "tags", tags)
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
	slog.Debug("filtered by create mod version", "count", len(result), "createVersion", createVersion)
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
	slog.Debug("filtered by minecraft version", "count", len(result), "minecraftVersion", minecraftVersion)
	// Hide paid
	if hidePaid {
		filtered := make([]schematicIndex, 0, len(result))
		for i := range result {
			if !result[i].Paid {
				filtered = append(filtered, result[i])
			}
		}
		result = filtered
	}
	slog.Debug("filtered by paid", "count", len(result), "hidePaid", hidePaid)
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
			slog.Error("error for bleve search query", "error", err.Error())
		}
		if searchResult != nil {
			count, err := s.bleveIndex.DocCount()
			slog.Debug("bleve search results", "total", searchResult.Total, "hits", len(searchResult.Hits), "stats", s.bleveIndex.StatsMap(), "index", count, "error", err)
			for _, si := range searchResult.Hits {
				for i := range result {
					if result[i].ID == si.ID {
						newResult = append(newResult, result[i])
					}
				}
			}
		}
		result = newResult
		slog.Debug("filtered by bleve", "count", len(result))
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
	slog.Debug("sorted", "count", len(result))

	ids := make([]string, len(result))
	for i := range result {
		ids[i] = result[i].ID
	}
	slog.Debug("returning ids", "count", len(ids))
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
	if scores == nil {
		// No trending data available; fall back to newest first
		return newestSort(a, b)
	}
	sa := scores[a.ID]
	sb := scores[b.ID]
	if sa > sb {
		return -1
	}
	if sa < sb {
		return 1
	}
	// Equal scores: break tie by newest first
	return newestSort(a, b)
}

// BuildIndex takes a set of schematics and rebuilds both the in-memory filter
// index and the Bleve full-text index. After building, it uploads a compressed
// cache snapshot to storage so subsequent pod starts can warm from it.
func (s *Service) BuildIndex(schematics []models.Schematic) {
	idx := newBleveIndex()
	batch := idx.NewBatch()
	filterIndex := make([]schematicIndex, len(schematics))
	cacheEntries := make([]indexCacheEntry, len(schematics))

	for i := range schematics {
		authorName := ""
		if schematics[i].Author != nil {
			authorName = schematics[i].Author.Username
		}
		si := schematicIndex{
			ID:          schematics[i].ID,
			Title:       stripHtmlRegex(schematics[i].Title),
			Description: stripHtmlRegex(schematics[i].Content),
			Created:     schematics[i].Created,
			Views:       int64(schematics[i].Views),
			Author:      authorName,
		}
		if parsedFloat, err := strconv.ParseFloat(schematics[i].Rating, 64); err == nil {
			si.Rating = parsedFloat
		}
		for _, c := range schematics[i].Categories {
			si.Categories = append(si.Categories, c.Name)
		}
		for _, t := range schematics[i].Tags {
			si.Tags = append(si.Tags, t.Name)
		}

		bi := bleveIndex{
			Title:         si.Title,
			Description:   si.Description,
			AIDescription: stripHtmlRegex(schematics[i].AIDescription),
			Tags:          si.Tags,
			Categories:    si.Categories,
			Author:        si.Author,
		}

		si.MinecraftVersion = schematics[i].MinecraftVersion
		si.CreateVersion = schematics[i].CreatemodVersion
		si.Paid = schematics[i].Paid

		if err := batch.Index(si.ID, bi); err != nil {
			slog.Error("bleve add index", "error", err.Error())
		}

		filterIndex[i] = si
		cacheEntries[i] = indexCacheEntry{SI: si, BI: bi}
	}

	if err := idx.Batch(batch); err != nil {
		slog.Error("bleve search batching", "error", err.Error())
		return
	}

	// Swap in the new index atomically.
	oldIdx := s.bleveIndex
	s.bleveIndex = idx
	s.index = filterIndex
	if oldIdx != nil {
		_ = oldIdx.Close()
	}

	// Persist cache to storage in the background.
	go s.saveCacheToStorage(cacheEntries)
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

// ---------------------------------------------------------------------------
// Storage-backed index cache (direct S3 via storage.Service)
// ---------------------------------------------------------------------------

// saveCacheToStorage serializes the index data as gzip-compressed JSON and
// uploads it to S3.
func (s *Service) saveCacheToStorage(entries []indexCacheEntry) {
	if s.storage == nil {
		slog.Warn("search: storage service not configured, skipping cache save")
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		slog.Warn("search: failed to marshal index cache", "error", err)
		return
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		slog.Warn("search: failed to gzip index cache", "error", err)
		return
	}
	if err := gz.Close(); err != nil {
		slog.Warn("search: failed to close gzip writer", "error", err)
		return
	}

	if err := s.storage.UploadRawBytes(context.Background(), cacheKey, buf.Bytes(), "application/gzip"); err != nil {
		slog.Warn("search: failed to upload index cache", "error", err)
		return
	}

	slog.Info("search: uploaded index cache to storage", "entries", len(entries), "bytes", buf.Len())
}

// loadCacheFromStorage downloads the compressed index cache from S3 and
// populates both the in-memory filter index and the Bleve full-text index.
func (s *Service) loadCacheFromStorage() error {
	if s.storage == nil {
		return fmt.Errorf("storage service not configured")
	}

	reader, err := s.storage.DownloadRaw(context.Background(), cacheKey)
	if err != nil {
		return err
	}
	defer reader.Close()

	gz, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return err
	}

	var entries []indexCacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	// Rebuild both indices from the cached data.
	idx := newBleveIndex()
	batch := idx.NewBatch()
	filterIndex := make([]schematicIndex, len(entries))

	for i, e := range entries {
		filterIndex[i] = e.SI
		if err := batch.Index(e.SI.ID, e.BI); err != nil {
			slog.Error("search: bleve index from cache", "error", err)
		}
	}

	if err := idx.Batch(batch); err != nil {
		slog.Error("search: bleve batch from cache", "error", err)
		return err
	}

	oldIdx := s.bleveIndex
	s.bleveIndex = idx
	s.index = filterIndex
	if oldIdx != nil {
		_ = oldIdx.Close()
	}

	return nil
}
