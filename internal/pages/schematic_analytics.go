package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type SchematicAnalyticsData struct {
	DefaultData
	SchematicName        string
	CommentCount         int64
	TotalViews           int
	TotalDownloads       int
	VDRatio              float64
	VDRatioPercent       string
	SiteAvgVDRatio       float64
	SiteAvgVDRatioPercent string
	VDRatioBetter        bool
	ShowNewFeatureBanner bool
	ViewsJSON            string
	DownloadsJSON        string
	VideoPlaysJSON       string
	YTClicksJSON         string
	TimeOnPageJSON       string
	LayerViewsJSON       string
}

func SchematicAnalyticsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	tmpl := []string{"template/include/schematic-analytics.html"}

	return func(e *server.RequestEvent) error {
		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("", nil)
		}

		name := chi.URLParam(e.Request, "name")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || schem == nil {
			return e.NotFoundError("schematic not found", nil)
		}

		if schem.AuthorID != userID {
			return e.ForbiddenError("not authorized", nil)
		}

		since := time.Now().UTC().AddDate(0, 0, -30)

		views, _ := appStore.Stats.HourlySchematicViews(ctx, schem.ID, since)
		downloads, _ := appStore.Stats.HourlySchematicDownloads(ctx, schem.ID, since)
		videoPlays, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventVideoPlay, since)
		ytClicks, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventYouTubeClick, since)
		timeOnPage, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventTimeOnPage, since)
		layerViews, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventLayerViewer, since)

		commentCount, _ := appStore.Comments.CountBySchematic(ctx, schem.ID)

		var totalViews, totalDownloads int
		for _, v := range views {
			totalViews += int(v.Count)
		}
		for _, d := range downloads {
			totalDownloads += int(d.Count)
		}

		var vdRatio float64
		if totalViews > 0 {
			vdRatio = float64(totalDownloads) / float64(totalViews)
		}

		var siteAvg float64
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio"); ok {
			siteAvg = cached
		} else {
			siteAvg, _ = appStore.Stats.GetSiteAvgVDRatio(ctx)
			cacheService.SetFloat("site_avg_vd_ratio", siteAvg)
		}

		cutoff := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

		d := SchematicAnalyticsData{
			SchematicName:         name,
			CommentCount:          commentCount,
			TotalViews:            totalViews,
			TotalDownloads:        totalDownloads,
			VDRatio:               vdRatio,
			VDRatioPercent:        fmt.Sprintf("%.2f%%", vdRatio*100),
			SiteAvgVDRatio:        siteAvg,
			SiteAvgVDRatioPercent: fmt.Sprintf("%.2f%%", siteAvg*100),
			VDRatioBetter:         vdRatio >= siteAvg,
			ShowNewFeatureBanner: schem.Created.Before(cutoff),
			ViewsJSON:            hourlyStatsJSON(views),
			DownloadsJSON:        hourlyStatsJSON(downloads),
			VideoPlaysJSON:       hourlyStatsJSON(videoPlays),
			YTClicksJSON:        hourlyStatsJSON(ytClicks),
			TimeOnPageJSON:       hourlyStatsJSON(timeOnPage),
			LayerViewsJSON:       hourlyStatsJSON(layerViews),
		}
		d.Populate(e)

		html, err := registry.LoadFiles(tmpl...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func hourlyStatsJSON(stats []store.HourlyStat) string {
	type point struct {
		X string `json:"x"`
		Y int64  `json:"y"`
	}
	pts := make([]point, len(stats))
	for i, s := range stats {
		pts[i] = point{X: s.Hour, Y: s.Count}
	}
	b, _ := json.Marshal(pts)
	return string(b)
}
