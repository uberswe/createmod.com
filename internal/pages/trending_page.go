package pages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"

	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
)

var trendingPageTemplates = append([]string{
	"./template/trending.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

var highestScoresPageTemplates = append([]string{
	"./template/highest_scores.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

const feedPageSize = 24

type FeedPageData struct {
	DefaultData
	Schematics []models.Schematic
	Page       int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
}

func TrendingPageHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		page = clampPage(page, 1000)

		limit := feedPageSize + 1
		offset := (page - 1) * feedPageSize

		all := getAllTrendingSchematicsFromStore(appStore, cacheService)
		var schematics []models.Schematic
		hasNext := false
		if offset < len(all) {
			end := offset + limit
			if end > len(all) {
				end = len(all)
			}
			schematics = all[offset:end]
		}
		if len(schematics) > feedPageSize {
			hasNext = true
			schematics = schematics[:feedPageSize]
		}

		translateSchematicTitles(schematics, translationService, cacheService, detectLanguageFromRequest(e.Request))

		d := FeedPageData{
			Schematics: schematics,
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/trending?p=%d", page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/trending?p=%d", page+1)
		}

		d.Populate(e)
		d.Title = "Trending Schematics"
		d.Description = "Discover the most popular Create mod schematics trending right now. Updated daily with the community's hottest builds, contraptions, and designs."
		d.Slug = "/trending"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ogURL := getTrendingOGImage(cacheService)
		if ogURL != "" {
			d.Thumbnail = ogURL
		}

		html, err := registry.LoadFiles(trendingPageTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func HighestScoresPageHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		page := 1
		if p := e.Request.URL.Query().Get("p"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		page = clampPage(page, 1000)

		limit := feedPageSize + 1
		offset := (page - 1) * feedPageSize

		results, err := appStore.Schematics.ListHighestRated(context.Background(), limit, offset)
		if err != nil {
			return err
		}
		hasNext := len(results) > feedPageSize
		if hasNext {
			results = results[:feedPageSize]
		}

		schematics := MapStoreSchematics(appStore, results, cacheService)
		translateSchematicTitles(schematics, translationService, cacheService, detectLanguageFromRequest(e.Request))

		d := FeedPageData{
			Schematics: schematics,
			Page:       page,
			HasPrev:    page > 1,
			HasNext:    hasNext,
		}
		if d.HasPrev {
			d.PrevURL = fmt.Sprintf("/highest-scores?p=%d", page-1)
		}
		if d.HasNext {
			d.NextURL = fmt.Sprintf("/highest-scores?p=%d", page+1)
		}

		d.Populate(e)
		d.Title = "Highest Rated Schematics"
		d.Description = "Browse the highest rated Create mod schematics as voted by the community. The best builds, machines, and contraptions ranked by user ratings."
		d.Slug = "/highest-scores"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ogURL := getHighestScoresOGImage(cacheService)
		if ogURL != "" {
			d.Thumbnail = ogURL
		}

		html, err := registry.LoadFiles(highestScoresPageTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

const (
	trendingOGCacheKey      = "og:trending_collage"
	highestScoresOGCacheKey = "og:highest_scores_collage"
)

func getTrendingOGImage(cacheService *cache.Service) string {
	if v, ok := cacheService.GetString(trendingOGCacheKey); ok {
		return v
	}
	return ""
}

func getHighestScoresOGImage(cacheService *cache.Service) string {
	if v, ok := cacheService.GetString(highestScoresOGCacheKey); ok {
		return v
	}
	return ""
}

func GenerateFeedOGImage(storageSvc *storage.Service, appStore *store.Store, cacheService *cache.Service, feedType string) {
	if storageSvc == nil {
		return
	}

	ctx := context.Background()
	var schematics []store.Schematic
	var err error

	switch feedType {
	case "trending":
		all := getAllTrendingSchematicsFromStore(appStore, cacheService)
		for i, s := range all {
			if i >= 4 {
				break
			}
			schematics = append(schematics, store.Schematic{
				ID:            s.ID,
				FeaturedImage: s.FeaturedImage,
			})
		}
	case "highest_scores":
		schematics, err = appStore.Schematics.ListHighestRated(ctx, 4, 0)
		if err != nil {
			slog.Warn("feed OG: failed to list highest rated", "error", err)
			return
		}
	default:
		return
	}

	if len(schematics) == 0 {
		return
	}

	var images []image.Image
	for _, s := range schematics {
		if s.FeaturedImage == "" {
			continue
		}
		key := fmt.Sprintf("%s/%s/%s", storage.CollectionPrefix("schematics"), s.ID, s.FeaturedImage)
		rc, err := storageSvc.DownloadRaw(ctx, key)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		img, err := imgconv.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		images = append(images, img)
		if len(images) >= 4 {
			break
		}
	}

	if len(images) == 0 {
		return
	}

	const (
		collageW = 800
		collageH = 450
	)

	dst := image.NewRGBA(image.Rect(0, 0, collageW, collageH))

	switch len(images) {
	case 1:
		draw.CatmullRom.Scale(dst, dst.Bounds(), images[0], images[0].Bounds(), draw.Over, nil)
	case 2:
		halfW := collageW / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, collageH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, collageH), images[1], images[1].Bounds(), draw.Over, nil)
	case 3:
		halfW := collageW / 2
		halfH := collageH / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, collageH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, halfH), images[1], images[1].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, halfH, collageW, collageH), images[2], images[2].Bounds(), draw.Over, nil)
	default:
		halfW := collageW / 2
		halfH := collageH / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, halfH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, halfH), images[1], images[1].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(0, halfH, halfW, collageH), images[2], images[2].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, halfH, collageW, collageH), images[3], images[3].Bounds(), draw.Over, nil)
	}

	var out bytes.Buffer
	bw := bufio.NewWriter(&out)
	if err := encodeWebP(bw, dst); err != nil {
		slog.Error("feed OG: failed to encode", "error", err)
		return
	}
	_ = bw.Flush()

	imageID := generateImageID()
	filename := feedType + "_og.webp"
	if err := storageSvc.UploadBytes(ctx, "images", imageID, filename, out.Bytes(), "image/webp"); err != nil {
		slog.Error("feed OG: failed to upload", "error", err)
		return
	}

	ogURL := fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)

	switch feedType {
	case "trending":
		cacheService.Set(trendingOGCacheKey, ogURL)
	case "highest_scores":
		cacheService.Set(highestScoresOGCacheKey, ogURL)
	}

	slog.Info("feed OG: generated", "type", feedType, "url", ogURL)
}
