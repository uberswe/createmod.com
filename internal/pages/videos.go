package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"createmod/internal/server"
)

var videosTemplates = append([]string{
	"./template/videos.html",
}, commonTemplates...)

// VideoItem represents a unique YouTube video referenced by a schematic.
type VideoItem struct {
	ID           string // YouTube ID
	Title        string
	ThumbnailURL string
	VideoURL     string
	SchematicURL string
}

type VideosData struct {
	DefaultData
	Videos   []VideoItem
	Page     int
	PageSize int
	HasPrev  bool
	HasNext  bool
	PrevURL  string
	NextURL  string
	Query    string
}

const videosCacheKey = "videos:trending"

var (
	// regexes to catch common YouTube URL forms
	reWatch  = regexp.MustCompile(`(?:v=)([A-Za-z0-9_-]{6,})`)
	reShort  = regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]{6,})`)
	reShorts = regexp.MustCompile(`youtube\.com/shorts/([A-Za-z0-9_-]{6,})`)
)

func youtubeID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Parse to normalize query when possible
	if u, err := url.Parse(s); err == nil && u != nil {
		// try query v
		if v := u.Query().Get("v"); v != "" {
			return v
		}
	}
	if m := reWatch.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if m := reShort.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	if m := reShorts.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	return ""
}

func youtubeThumb(id string) string {
	if id == "" {
		return ""
	}
	// mqdefault is 320x180 (16:9) — avoids black bars that hqdefault (4:3) causes
	return "https://i.ytimg.com/vi/" + id + "/mqdefault.jpg"
}

// computeTrendingVideos fetches schematics with videos and sorts them by
// trending score using real engagement data from aggregate tables.
func computeTrendingVideos(appStore *store.Store) []VideoItem {
	schematics, err := appStore.Schematics.ListApprovedWithVideo(context.Background(), 500, 0)
	if err != nil {
		return nil
	}

	engagement, _ := appStore.ViewRatings.FetchTrendingData(context.Background(), 7)

	type scoredVideo struct {
		item  VideoItem
		score float64
	}
	seen := make(map[string]int)
	scoredItems := make([]scoredVideo, 0, len(schematics))
	for _, s := range schematics {
		vid := youtubeID(s.Video)
		if vid == "" {
			continue
		}

		var score float64
		if engagement != nil {
			score = trendingScore(s.Created, engagement.RecentViews[s.ID], engagement.TotalViews[s.ID], engagement.RatingCount[s.ID], engagement.RatingSum[s.ID], engagement.RecentDownloads[s.ID], engagement.TotalDownloads[s.ID])
		} else {
			score = s.Created.Sub(trendingEpoch).Seconds() / trendingTimescale
		}

		if idx, exists := seen[vid]; exists {
			if score > scoredItems[idx].score {
				scoredItems[idx] = scoredVideo{
					item: VideoItem{
						ID:           vid,
						Title:        s.Title,
						ThumbnailURL: youtubeThumb(vid),
						VideoURL:     "https://www.youtube.com/watch?v=" + vid,
						SchematicURL: "/schematics/" + s.Name,
					},
					score: score,
				}
			}
			continue
		}
		seen[vid] = len(scoredItems)
		scoredItems = append(scoredItems, scoredVideo{
			item: VideoItem{
				ID:           vid,
				Title:        s.Title,
				ThumbnailURL: youtubeThumb(vid),
				VideoURL:     "https://www.youtube.com/watch?v=" + vid,
				SchematicURL: "/schematics/" + s.Name,
			},
			score: score,
		})
	}

	sort.SliceStable(scoredItems, func(i, j int) bool {
		return scoredItems[i].score > scoredItems[j].score
	})

	items := make([]VideoItem, 0, len(scoredItems))
	for _, sv := range scoredItems {
		items = append(items, sv.item)
	}
	return items
}

// deadVideoCacheKey returns the cache key used to remember that a YouTube
// video ID has been confirmed unavailable.
func deadVideoCacheKey(videoID string) string {
	return "yt_dead:" + videoID
}

// ytCheckClient is a shared HTTP client for YouTube availability checks
// with a short timeout so a single slow response doesn't stall warming.
var ytCheckClient = &http.Client{Timeout: 10 * time.Second}

// isYouTubeAvailable checks whether a YouTube video is still publicly
// accessible by hitting the oEmbed endpoint (no API key required).
// Returns true if the video is available, false if removed/private.
func isYouTubeAvailable(videoID string) bool {
	oembedURL := "https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=" + url.QueryEscape(videoID) + "&format=json"
	resp, err := ytCheckClient.Get(oembedURL)
	if err != nil {
		// Network error — assume available to avoid false positives
		slog.Debug("videos: YouTube oEmbed request failed", "id", videoID, "error", err)
		return true
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// filterAvailableVideos removes videos that YouTube reports as unavailable.
// It caches dead video IDs for 24 hours so subsequent warming cycles skip
// the HTTP check. A 100ms delay between checks avoids throttling.
func filterAvailableVideos(items []VideoItem, cacheService *cache.Service) []VideoItem {
	available := make([]VideoItem, 0, len(items))
	checked := 0
	removed := 0
	for _, item := range items {
		// Check if we already know this video is dead
		if _, dead := cacheService.Get(deadVideoCacheKey(item.ID)); dead {
			removed++
			continue
		}

		// Rate-limit: small delay between outbound checks
		if checked > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		checked++

		if isYouTubeAvailable(item.ID) {
			available = append(available, item)
		} else {
			slog.Info("videos: removing unavailable YouTube video", "id", item.ID, "title", item.Title)
			cacheService.SetWithTTL(deadVideoCacheKey(item.ID), true, 24*time.Hour)
			removed++
		}
	}
	if removed > 0 {
		slog.Info("videos: filtered unavailable videos", "removed", removed, "checked", checked, "remaining", len(available))
	}
	return available
}

// WarmVideosCache precomputes the trending videos list, filters out
// unavailable YouTube videos, and stores the result in cache so no user
// request has to pay the cost. Called at startup and periodically from
// a background ticker.
func WarmVideosCache(cacheService *cache.Service, appStore *store.Store) {
	slog.Debug("Warming videos page cache")
	items := computeTrendingVideos(appStore)
	if items != nil {
		items = filterAvailableVideos(items, cacheService)
		cacheService.Set(videosCacheKey, items)
	}
	slog.Debug("Videos page cache warmed", "count", len(items))
}

// getCachedVideos returns the cached trending videos list, computing it on
// cache miss (should only happen if the warm function hasn't run yet).
func getCachedVideos(appStore *store.Store, cacheService *cache.Service) []VideoItem {
	if cached, ok := cacheService.Get(videosCacheKey); ok {
		if items, ok := cached.([]VideoItem); ok {
			return items
		}
	}
	// Cache miss — compute and store
	items := computeTrendingVideos(appStore)
	if items != nil {
		cacheService.Set(videosCacheKey, items)
	}
	return items
}

// VideosHandler renders a page of unique YouTube videos referenced by schematics,
// sorted by trending score. Reads from a preemptively warmed cache.
func VideosHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Pagination params
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		pageSize := 9
		// Query filter
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		qLower := strings.ToLower(q)

		allItems := getCachedVideos(appStore, cacheService)

		// Apply search filter
		var items []VideoItem
		if q == "" {
			items = allItems
		} else {
			items = make([]VideoItem, 0, len(allItems))
			for _, it := range allItems {
				if strings.Contains(strings.ToLower(it.Title), qLower) {
					items = append(items, it)
				}
			}
		}

		// Apply pagination
		start := (page - 1) * pageSize
		if start > len(items) {
			start = len(items)
		}
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		paged := items[start:end]
		hasPrev := page > 1
		hasNext := len(items) > end

		d := VideosData{
			Videos:   paged,
			Page:     page,
			PageSize: pageSize,
			HasPrev:  hasPrev,
			HasNext:  hasNext,
			Query:    q,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/videos?p=%d", page-1)
			if q != "" {
				d.PrevURL += "&q=" + url.QueryEscape(q)
			}
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/videos?p=%d", page+1)
			if q != "" {
				d.NextURL += "&q=" + url.QueryEscape(q)
			}
		}

		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Create Mod Videos"))
		d.Title = i18n.T(d.Language, "Create Mod Videos")
		d.Description = i18n.T(d.Language, "Videos from published schematics")
		d.Slug = "/videos"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		html, err := registry.LoadFiles(videosTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
