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
	"os"
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
	// An environment scope and a date suffix (YYYY-MM-DD) are appended to form
	// the full key: the scope because dev and prod share one bucket and must
	// not overwrite each other's snapshots (pods reload this cache every 10
	// minutes, so a shared key lets one environment poison the other's
	// in-memory index); the date so Minio/S3 bucket versioning doesn't
	// accumulate thousands of old versions under a single key.
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
	Downloads        int64
	Rating           float64
	RatingCount      int
	Author           string
	MinecraftVersion string
	CreateVersion    string
	BlockNames       []string
	ModNames         []string
	BlockCount       int
	DimX             int
	DimY             int
	DimZ             int
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

// GetTrendingScores returns the current trending scores map.
func (s *Service) GetTrendingScores() map[string]float64 {
	return s.trendingScores
}

// GetIndexForIDs returns index entries matching the given IDs.
func (s *Service) GetIndexForIDs(ids []string) []schematicIndex {
	if s.index == nil {
		return nil
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	var result []schematicIndex
	for _, si := range s.index {
		if _, ok := idSet[si.ID]; ok {
			result = append(result, si)
		}
	}
	return result
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
			Downloads:   int64(schematics[i].Downloads),
			RatingCount: schematics[i].RatingCount,
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
		si.BlockCount = schematics[i].BlockCount
		si.DimX = schematics[i].DimX
		si.DimY = schematics[i].DimY
		si.DimZ = schematics[i].DimZ

		filterIndex[i] = si
	}

	s.index = filterIndex

	// Persist cache to storage in the background.
	go s.saveCacheToStorage(filterIndex)
}

// GetIndex returns the in-memory filter index for use by Meilisearch sync.
func (s *Service) GetIndex() []schematicIndex {
	return s.index
}

// FilterMaxStats holds the global maximum values for slider-based search filters.
type FilterMaxStats struct {
	BlockCount int
	DimX       int
	DimY       int
	DimZ       int
}

const (
	sliderMaxBlockCount = 50000
	sliderMaxDimX       = 150
	sliderMaxDimY       = 100
	sliderMaxDimZ       = 150
)

// MaxStats returns capped maximum values for slider upper bounds in the search UI.
// Raw maximums are capped to cover ~P99 of the data so sliders remain usable.
func (s *Service) MaxStats() FilterMaxStats {
	var stats FilterMaxStats
	for _, si := range s.index {
		if si.BlockCount > stats.BlockCount {
			stats.BlockCount = si.BlockCount
		}
		if si.DimX > stats.DimX {
			stats.DimX = si.DimX
		}
		if si.DimY > stats.DimY {
			stats.DimY = si.DimY
		}
		if si.DimZ > stats.DimZ {
			stats.DimZ = si.DimZ
		}
	}
	if stats.BlockCount > sliderMaxBlockCount {
		stats.BlockCount = sliderMaxBlockCount
	}
	if stats.DimX > sliderMaxDimX {
		stats.DimX = sliderMaxDimX
	}
	if stats.DimY > sliderMaxDimY {
		stats.DimY = sliderMaxDimY
	}
	if stats.DimZ > sliderMaxDimZ {
		stats.DimZ = sliderMaxDimZ
	}
	return stats
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

// cacheEnvScope returns the environment segment of the cache key, from the
// ENVIRONMENT env var ("production", "dev"). Empty means a local run.
func cacheEnvScope() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "local"
}

// cacheKeyForDate returns the environment- and date-stamped cache key for the
// given time.
func cacheKeyForDate(t time.Time) string {
	return cacheKeyPrefix + cacheEnvScope() + "_" + t.UTC().Format("2006-01-02") + cacheKeySuffix
}

// unscopedCacheKeyForDate returns the pre-environment-scoping key. Kept for
// migration: it is tried as a load fallback (so pods deployed before their
// environment has saved a scoped snapshot still warm up) and deleted after a
// successful save.
func unscopedCacheKeyForDate(t time.Time) string {
	return cacheKeyPrefix + t.UTC().Format("2006-01-02") + cacheKeySuffix
}

// saveCacheToStorage serializes the index data as gzip-compressed JSON and
// uploads it to S3 using a date-stamped key. After a successful upload it
// removes the previous day's key and the legacy unversioned key to prevent
// unbounded version accumulation in versioned S3 buckets.
func (s *Service) saveCacheToStorage(index []schematicIndex) {
	if s.storage == nil {
		slog.Warn("search: storage service not configured, skipping cache save")
		return
	}

	data, err := json.Marshal(index)
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

	slog.Info("search: uploaded index cache to storage", "key", todayKey, "entries", len(index), "uncompressed_bytes", len(data), "compressed_bytes", buf.Len())

	// Clean up previous day's cache key so old versions don't accumulate.
	yesterdayKey := cacheKeyForDate(now.AddDate(0, 0, -1))
	if yesterdayKey != todayKey {
		if err := s.storage.DeleteRaw(context.Background(), yesterdayKey); err != nil {
			slog.Debug("search: failed to delete yesterday's cache (may not exist)", "key", yesterdayKey, "error", err)
		}
	}

	// Remove pre-environment-scoping and legacy keys (migration cleanup).
	// The unscoped key was shared between dev and prod, letting one
	// environment's snapshot poison the other's in-memory index.
	for _, key := range []string{
		unscopedCacheKeyForDate(now),
		unscopedCacheKeyForDate(now.AddDate(0, 0, -1)),
		legacyCacheKey,
	} {
		if err := s.storage.DeleteRaw(context.Background(), key); err != nil {
			slog.Debug("search: failed to delete old cache key (may not exist)", "key", key, "error", err)
		}
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
		unscopedCacheKeyForDate(now),
		unscopedCacheKeyForDate(now.AddDate(0, 0, -1)),
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

	var index []schematicIndex
	if err := json.Unmarshal(data, &index); err != nil {
		// Fall back to legacy format with wrapper struct.
		type legacyCacheEntry struct {
			SI schematicIndex `json:"si"`
		}
		var legacy []legacyCacheEntry
		if err2 := json.Unmarshal(data, &legacy); err2 != nil {
			return err
		}
		index = make([]schematicIndex, len(legacy))
		for i, e := range legacy {
			index[i] = e.SI
		}
	}

	if len(index) == 0 {
		return nil
	}

	s.index = index
	return nil
}
