package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

var schematicStatsTemplates = append([]string{
	"./template/schematic-stats.html",
}, commonTemplates...)

type SchematicAnalyticsData struct {
	DefaultData
	SchematicName         string
	SchematicTitle        string
	CommentCount          int64
	TotalViews            int
	TotalDownloads        int
	VDRatioPercent        string
	SiteAvgVDRatioPercent string
	VDRatioBetter         bool
	ShowNewFeatureBanner  bool
	HasVideo              bool
	ViewsJSON             template.JS
	DownloadsJSON         template.JS
	VideoPlaysJSON        template.JS
	YTClicksJSON          template.JS
	TimeOnPageJSON        template.JS
	LayerViewsJSON        template.JS
}

func SchematicStatsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
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
		hasVideo := schem.Video != ""

		views, _ := appStore.Stats.HourlySchematicViews(ctx, schem.ID, since)
		downloads, _ := appStore.Stats.HourlySchematicDownloads(ctx, schem.ID, since)
		var videoPlays, ytClicks []store.HourlyStat
		if hasVideo {
			videoPlays, _ = appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventVideoPlay, since)
			ytClicks, _ = appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventYouTubeClick, since)
		}
		timeOnPage, _ := appStore.Stats.HourlySchematicEvents(ctx, schem.ID, store.EventTimeOnPage, since)
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
		if cached, ok := cacheService.GetFloat("site_avg_vd_ratio"); ok {
			siteAvg = cached
		} else {
			siteAvg, _ = appStore.Stats.GetSiteAvgVDRatio(ctx)
			cacheService.SetFloat("site_avg_vd_ratio", siteAvg)
		}

		cutoff := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)

		d := SchematicAnalyticsData{
			SchematicName:         name,
			SchematicTitle:        schem.Title,
			CommentCount:          commentCount,
			TotalViews:            totalViews,
			TotalDownloads:        totalDownloads,
			VDRatioPercent:        fmt.Sprintf("%.2f%%", vdRatio*100),
			SiteAvgVDRatioPercent: fmt.Sprintf("%.2f%%", siteAvg*100),
			VDRatioBetter:         vdRatio >= siteAvg,
			ShowNewFeatureBanner:  schem.Created.Before(cutoff),
			HasVideo:              hasVideo,
			ViewsJSON:             template.JS(hourlyStatsJSON(views)),
			DownloadsJSON:         template.JS(hourlyStatsJSON(downloads)),
			VideoPlaysJSON:        template.JS(hourlyStatsJSON(videoPlays)),
			YTClicksJSON:          template.JS(hourlyStatsJSON(ytClicks)),
			TimeOnPageJSON:        template.JS(hourlyStatsJSON(timeOnPage)),
			LayerViewsJSON:        template.JS(hourlyStatsJSON(layerViews)),
		}
		d.Populate(e)
		d.Title = fmt.Sprintf("%s - %s", i18n.T(d.Language, "Stats"), schem.Title)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, schem.Title, "/schematics/"+name, i18n.T(d.Language, "Stats"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		out, err := registry.LoadFiles(schematicStatsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, out)
	}
}

func hourlyStatsJSON(stats []store.HourlyStat) string {
	type point struct {
		X string `json:"x"`
		Y int64  `json:"y"`
	}

	lookup := make(map[string]int64, len(stats))
	for _, s := range stats {
		lookup[s.Hour] = s.Count
	}

	now := time.Now().UTC().Truncate(time.Hour)
	start := now.AddDate(0, 0, -30)
	totalHours := int(now.Sub(start).Hours()) + 1

	pts := make([]point, 0, totalHours)
	for t := start; !t.After(now); t = t.Add(time.Hour) {
		key := t.Format("2006-01-02 15")
		pts = append(pts, point{X: key, Y: lookup[key]})
	}

	b, _ := json.Marshal(pts)
	return string(b)
}
