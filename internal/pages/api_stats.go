package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type apiHourlyStat struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type apiSchematicStatsResponse struct {
	Name           string          `json:"name"`
	Title          string          `json:"title"`
	TotalViews     int             `json:"total_views"`
	TotalDownloads int             `json:"total_downloads"`
	Comments       int64           `json:"comments"`
	VDRatio        float64         `json:"vd_ratio"`
	SiteAvgVDRatio float64         `json:"site_avg_vd_ratio"`
	HasVideo       bool            `json:"has_video"`
	Views          []apiHourlyStat `json:"views"`
	Downloads      []apiHourlyStat `json:"downloads"`
	VideoPlays     []apiHourlyStat `json:"video_plays,omitempty"`
	YTClicks       []apiHourlyStat `json:"yt_clicks,omitempty"`
	TimeOnPage     []apiHourlyStat `json:"time_on_page"`
	LayerViews     []apiHourlyStat `json:"layer_views"`
}

type apiUserStatsSchematic struct {
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	FeaturedImage string    `json:"featured_image"`
	Views         int       `json:"views"`
	Downloads     int       `json:"downloads"`
	VDRatio       float64   `json:"vd_ratio"`
	Created       time.Time `json:"created"`
}

type apiUserStatsResponse struct {
	TotalViews     int                     `json:"total_views"`
	TotalDownloads int                     `json:"total_downloads"`
	VDRatio        float64                 `json:"vd_ratio"`
	SiteAvgVDRatio float64                 `json:"site_avg_vd_ratio"`
	Views          []apiHourlyStat         `json:"views"`
	Downloads      []apiHourlyStat         `json:"downloads"`
	VideoPlays     []apiHourlyStat         `json:"video_plays"`
	YTClicks       []apiHourlyStat         `json:"yt_clicks"`
	TimeOnPage     []apiHourlyStat         `json:"time_on_page"`
	LayerViews     []apiHourlyStat         `json:"layer_views"`
	Schematics     []apiUserStatsSchematic `json:"schematics"`
	TotalSchematics int                    `json:"total_schematics"`
	Page            int                    `json:"page"`
	PageSize        int                    `json:"page_size"`
	HasNext         bool                   `json:"has_next"`
	HasPrev         bool                   `json:"has_prev"`
}

func fillHourlyStats(stats []store.HourlyStat) []apiHourlyStat {
	lookup := make(map[string]int64, len(stats))
	for _, s := range stats {
		lookup[s.Hour] = s.Count
	}

	now := time.Now().UTC().Truncate(time.Hour)
	start := now.AddDate(0, 0, -30)

	result := make([]apiHourlyStat, 0, 721)
	for t := start; !t.After(now); t = t.Add(time.Hour) {
		key := t.Format("2006-01-02 15")
		result = append(result, apiHourlyStat{Hour: key, Count: lookup[key]})
	}
	return result
}

func resolveAPIKeyUserID(appStore *store.Store, r *http.Request) (key *store.APIKey, ok bool) {
	plaintext := getAPIKeyFromRequest(r)
	if plaintext == "" {
		return nil, false
	}
	return verifyAPIKeyFromStore(appStore, plaintext)
}

// APISchematicStatsHandler serves GET /api/schematics/{name}/stats.
func APISchematicStatsHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}/stats"

		key, ok := resolveAPIKeyUserID(appStore, e.Request)
		if !ok {
			return writeJSON(e, http.StatusUnauthorized, map[string]string{
				"error": "API key required. Get one at /settings/api-keys",
			})
		}
		defer func() { recordAPIKeyUsageStore(appStore, key.ID, endpoint) }()
		if allowed, retry := rateLimitAllow(rl, key.ID, effectiveRateLimit(key, defaultAPIRateLimitPerMinute)); !allowed {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		name := e.Request.PathValue("name")
		if name == "" {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing schematic name"})
		}

		ctx := context.Background()
		schem, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || schem == nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic not found"})
		}
		if schem.AuthorID != key.UserID {
			return writeJSON(e, http.StatusForbidden, map[string]string{"error": "you do not own this schematic"})
		}

		since := time.Now().UTC().AddDate(0, 0, -30)
		hasVideo := schem.Video != ""

		views, _ := appStore.Stats.HourlySchematicViews(ctx, schem.ID, since)
		downloads, _ := appStore.Stats.HourlySchematicDownloads(ctx, schem.ID, since)
		var videoPlays, ytClicks []store.HourlyStat
		if hasVideo {
			videoPlays, _ = appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventVideoPlay, since)
			ytClicks, _ = appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventYouTubeClick, since)
		}
		timeOnPage, _ := appStore.Stats.HourlySchematicEventAvg(ctx, schem.ID, store.EventTimeOnPage, since)
		layerViews, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventLayerViewer, since)

		commentCount, _ := appStore.Comments.CountBySchematic(ctx, schem.ID)

		var totalViews, totalDownloads int
		for _, v := range views {
			totalViews += int(v.Count)
		}
		for _, dl := range downloads {
			totalDownloads += int(dl.Count)
		}

		var vdRatio float64
		if totalViews > 0 {
			vdRatio = float64(totalDownloads) / float64(totalViews)
		}

		var siteAvg float64
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio_v2"); ok {
			siteAvg = cached
		} else {
			siteAvg, _ = appStore.Stats.GetSiteAvgVDRatioSinceCutoff(ctx, HourlyTrackingCutoff)
			cacheService.SetFloat("site_avg_vd_ratio_v2", siteAvg)
		}

		resp := apiSchematicStatsResponse{
			Name:           name,
			Title:          schem.Title,
			TotalViews:     totalViews,
			TotalDownloads: totalDownloads,
			Comments:       commentCount,
			VDRatio:        vdRatio,
			SiteAvgVDRatio: siteAvg,
			HasVideo:       hasVideo,
			Views:          fillHourlyStats(views),
			Downloads:      fillHourlyStats(downloads),
			TimeOnPage:     fillHourlyStats(timeOnPage),
			LayerViews:     fillHourlyStats(layerViews),
		}
		if hasVideo {
			resp.VideoPlays = fillHourlyStats(videoPlays)
			resp.YTClicks = fillHourlyStats(ytClicks)
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}

// APIUserStatsHandler serves GET /api/user/stats.
func APIUserStatsHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/user/stats"

		key, ok := resolveAPIKeyUserID(appStore, e.Request)
		if !ok {
			return writeJSON(e, http.StatusUnauthorized, map[string]string{
				"error": "API key required. Get one at /settings/api-keys",
			})
		}
		defer func() { recordAPIKeyUsageStore(appStore, key.ID, endpoint) }()
		if allowed, retry := rateLimitAllow(rl, key.ID, effectiveRateLimit(key, defaultAPIRateLimitPerMinute)); !allowed {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		ctx := context.Background()
		since := time.Now().UTC().AddDate(0, 0, -30)

		hViews, _ := appStore.Stats.HourlyUserViews(ctx, key.UserID, since)
		hDownloads, _ := appStore.Stats.HourlyUserDownloads(ctx, key.UserID, since)
		hVideoPlays, _ := appStore.Stats.HourlyUserEvents(ctx, key.UserID, store.EventVideoPlay, since)
		hYTClicks, _ := appStore.Stats.HourlyUserEvents(ctx, key.UserID, store.EventYouTubeClick, since)
		hTimeOnPage, _ := appStore.Stats.HourlyUserEventAvg(ctx, key.UserID, store.EventTimeOnPage, since)
		hLayerViews, _ := appStore.Stats.HourlyUserEvents(ctx, key.UserID, store.EventLayerViewer, since)

		var totalViews, totalDownloads int
		for _, v := range hViews {
			totalViews += int(v.Count)
		}
		for _, dl := range hDownloads {
			totalDownloads += int(dl.Count)
		}

		var vdRatio float64
		if totalViews > 0 {
			vdRatio = float64(totalDownloads) / float64(totalViews)
		}

		var siteAvg float64
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio_v2"); ok {
			siteAvg = cached
		} else {
			siteAvg, _ = appStore.Stats.GetSiteAvgVDRatioSinceCutoff(ctx, HourlyTrackingCutoff)
			cacheService.SetFloat("site_avg_vd_ratio_v2", siteAvg)
		}

		pageSize := 20
		page := 1
		if v := e.Request.URL.Query().Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		offset := (page - 1) * pageSize

		totalSchematics, _ := appStore.Stats.CountUserSchematics(ctx, key.UserID)
		schematicStats, _ := appStore.Stats.ListSchematicStats(ctx, key.UserID, pageSize, offset)

		schematics := make([]apiUserStatsSchematic, 0, len(schematicStats))
		for _, s := range schematicStats {
			var ratio float64
			if s.Views > 0 {
				ratio = float64(s.Downloads) / float64(s.Views)
			}
			schematics = append(schematics, apiUserStatsSchematic{
				Name:          s.Name,
				Title:         s.Title,
				FeaturedImage: s.FeaturedImage,
				Views:         s.Views,
				Downloads:     s.Downloads,
				VDRatio:       ratio,
				Created:       s.Created,
			})
		}

		resp := apiUserStatsResponse{
			TotalViews:      totalViews,
			TotalDownloads:  totalDownloads,
			VDRatio:         vdRatio,
			SiteAvgVDRatio:  siteAvg,
			Views:           fillHourlyStats(hViews),
			Downloads:       fillHourlyStats(hDownloads),
			VideoPlays:      fillHourlyStats(hVideoPlays),
			YTClicks:        fillHourlyStats(hYTClicks),
			TimeOnPage:      fillHourlyStats(hTimeOnPage),
			LayerViews:      fillHourlyStats(hLayerViews),
			Schematics:      schematics,
			TotalSchematics: totalSchematics,
			Page:            page,
			PageSize:        pageSize,
			HasNext:         page*pageSize < totalSchematics,
			HasPrev:         page > 1,
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}
