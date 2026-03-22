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
	"strconv"
	"strings"
	"time"

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

	// cacheKeyPrefix is the storage path prefix for the serialized index cache.
	// A date suffix (YYYY-MM-DD) is appended to form the full key, so that
	// Minio/S3 bucket versioning doesn't accumulate thousands of old versions
	// under a single key.
	cacheKeyPrefix = "_internal/search_index_cache_"
	cacheKeySuffix = ".json.gz"

	// legacyCacheKey is the old unversioned key. Kept for migration: we try
	// loading from it as a fallback and delete it after a successful save.
	legacyCacheKey = "_internal/search_index_cache.json.gz"
)

type Service struct {
	index          []schematicIndex
	storage        *storage.Service
	trendingScores map[string]float64
}

type schematicIndex struct {
	ID               string
	Title            string
	Description      string
	AIDescription    string
	Created          time.Time
	Tags             []string
	Categories       []string
	Views            int64
	Rating           float64
	Author           string
	MinecraftVersion string
	CreateVersion    string
	Paid             bool
	BlockNames       []string
	ModNames         []string
}

// indexCacheEntry holds filter-index data for one schematic.
type indexCacheEntry struct {
	SI schematicIndex `json:"si"`
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

// NewEmpty creates a search service without loading the S3 cache, so the
// caller is not blocked by network I/O. The index starts empty and is
// populated by the River SearchIndexWorker which runs on start.
func NewEmpty(storageSvc *storage.Service) *Service {
	return &Service{storage: storageSvc}
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

// BuildIndex takes a set of schematics and rebuilds the in-memory filter
// index. After building, it uploads a compressed cache snapshot to storage
// so subsequent pod starts can warm from it.
func (s *Service) BuildIndex(schematics []models.Schematic, modDisplayNames map[string]string) {
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

		blockNames := ExtractBlockNames(schematics[i].Materials)
		si.BlockNames = blockNames

		// Resolve mod namespaces to display names.
		if modDisplayNames != nil {
			var modNames []string
			for _, ns := range schematics[i].Mods {
				if name, ok := modDisplayNames[ns]; ok && name != "" {
					modNames = append(modNames, name)
				}
			}
			si.ModNames = modNames
		}

		si.AIDescription = stripHtmlRegex(schematics[i].AIDescription)
		si.MinecraftVersion = schematics[i].MinecraftVersion
		si.CreateVersion = schematics[i].CreatemodVersion
		si.Paid = schematics[i].Paid

		filterIndex[i] = si
		cacheEntries[i] = indexCacheEntry{SI: si}
	}

	s.index = filterIndex

	// Persist cache to storage in the background.
	go s.saveCacheToStorage(cacheEntries)
}

// GetIndex returns the in-memory filter index for use by Meilisearch sync.
func (s *Service) GetIndex() []schematicIndex {
	return s.index
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

// materialEntry represents a single block entry in the Materials JSON.
type materialEntry struct {
	BlockID string `json:"block_id"`
	Count   int    `json:"count"`
}

// ExtractBlockNames parses the Materials JSON field (e.g. [{"block_id":"create:brass_casing","count":12}])
// and returns deduplicated, human-readable block names by stripping the namespace
// and title-casing the result (e.g. "Brass Casing").
func ExtractBlockNames(materialsJSON string) []string {
	if materialsJSON == "" {
		return nil
	}
	var entries []materialEntry
	if err := json.Unmarshal([]byte(materialsJSON), &entries); err != nil {
		return nil
	}
	if len(entries) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(entries))
	names := make([]string, 0, len(entries))
	caser := cases.Title(language.English)

	for _, e := range entries {
		blockID := e.BlockID
		// Strip namespace prefix (e.g. "create:" or "minecraft:")
		if idx := strings.LastIndex(blockID, ":"); idx >= 0 {
			blockID = blockID[idx+1:]
		}
		// Convert underscores to spaces and title-case
		name := caser.String(strings.ReplaceAll(blockID, "_", " "))
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func stripHtmlRegex(s string) string {
	r := regexp.MustCompile(regex)
	return r.ReplaceAllString(s, " ")
}

// ---------------------------------------------------------------------------
// Storage-backed index cache (direct S3 via storage.Service)
// ---------------------------------------------------------------------------

// cacheKeyForDate returns the date-stamped cache key for the given time.
func cacheKeyForDate(t time.Time) string {
	return cacheKeyPrefix + t.UTC().Format("2006-01-02") + cacheKeySuffix
}

// saveCacheToStorage serializes the index data as gzip-compressed JSON and
// uploads it to S3 using a date-stamped key. After a successful upload it
// removes the previous day's key and the legacy unversioned key to prevent
// unbounded version accumulation in versioned S3 buckets.
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

	now := time.Now()
	todayKey := cacheKeyForDate(now)

	if err := s.storage.UploadRawBytes(context.Background(), todayKey, buf.Bytes(), "application/gzip"); err != nil {
		slog.Warn("search: failed to upload index cache", "error", err)
		return
	}

	slog.Info("search: uploaded index cache to storage", "key", todayKey, "entries", len(entries), "bytes", buf.Len())

	// Clean up previous day's cache key so old versions don't accumulate.
	yesterdayKey := cacheKeyForDate(now.AddDate(0, 0, -1))
	if yesterdayKey != todayKey {
		if err := s.storage.DeleteRaw(context.Background(), yesterdayKey); err != nil {
			slog.Debug("search: failed to delete yesterday's cache (may not exist)", "key", yesterdayKey, "error", err)
		}
	}

	// Remove legacy unversioned key (one-time migration cleanup).
	if err := s.storage.DeleteRaw(context.Background(), legacyCacheKey); err != nil {
		slog.Debug("search: failed to delete legacy cache key (may not exist)", "error", err)
	}
}

// loadCacheFromStorage downloads the compressed index cache from S3 and
// populates the in-memory filter index.
// It tries today's date-stamped key first, then yesterday's, then the
// legacy unversioned key as a migration fallback.
func (s *Service) loadCacheFromStorage() error {
	if s.storage == nil {
		return fmt.Errorf("storage service not configured")
	}

	now := time.Now()
	keysToTry := []string{
		cacheKeyForDate(now),
		cacheKeyForDate(now.AddDate(0, 0, -1)),
		legacyCacheKey,
	}

	var reader io.ReadCloser
	var err error
	for _, key := range keysToTry {
		reader, err = s.storage.DownloadRaw(context.Background(), key)
		if err == nil {
			slog.Info("search: loading cache from storage", "key", key)
			break
		}
	}
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

	filterIndex := make([]schematicIndex, len(entries))
	for i, e := range entries {
		filterIndex[i] = e.SI
	}

	s.index = filterIndex
	return nil
}
