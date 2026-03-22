package pages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"createmod/internal/cache"
	"createmod/internal/storage"
	"createmod/internal/store"

	"github.com/sunshineplan/imgconv"

	_ "image/jpeg"
	_ "image/png"
)

// recoverYouTubeThumbnail fetches the YouTube video thumbnail for a schematic
// that has no featured image, converts it to WebP, uploads it to Minio, and
// updates the database record. Runs in a background goroutine and uses a cache
// key to prevent duplicate attempts.
func recoverYouTubeThumbnail(appStore *store.Store, storageSvc *storage.Service, cacheService *cache.Service, schematicID, videoID string) {
	// Deduplicate: only attempt once per schematic per cache TTL
	cacheKey := "yt_thumb_recover:" + schematicID
	if _, already := cacheService.Get(cacheKey); already {
		return
	}
	cacheService.SetWithTTL(cacheKey, true, 1*time.Hour)

	go func() {
		ctx := context.Background()

		// Re-check the schematic in case it was updated between page load and goroutine start
		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil || schem.FeaturedImage != "" {
			return
		}

		// Try YouTube thumbnail URLs in order of quality
		thumbURLs := []string{
			fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID),
			fmt.Sprintf("https://img.youtube.com/vi/%s/sddefault.jpg", videoID),
			fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", videoID),
		}

		var imgData []byte
		for _, u := range thumbURLs {
			resp, err := http.Get(u)
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			// YouTube returns a small grey placeholder for missing resolutions
			if resp.StatusCode == 200 && len(data) > 5000 {
				imgData = data
				break
			}
		}

		if imgData == nil {
			slog.Warn("youtube thumbnail recovery: no thumbnail available",
				"schematic_id", schematicID, "video_id", videoID)
			return
		}

		// Convert to WebP
		img, err := imgconv.Decode(bytes.NewReader(imgData))
		if err != nil {
			slog.Error("youtube thumbnail recovery: failed to decode image",
				"schematic_id", schematicID, "error", err)
			return
		}

		var out bytes.Buffer
		bw := bufio.NewWriter(&out)
		if err := imgconv.Write(bw, img, &imgconv.FormatOption{
			Format:       imgconv.WEBP,
			EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)},
		}); err != nil {
			slog.Error("youtube thumbnail recovery: failed to encode WebP",
				"schematic_id", schematicID, "error", err)
			return
		}
		_ = bw.Flush()

		filename := "youtube_thumbnail.webp"
		s3Collection := storage.CollectionPrefix("schematics")

		if err := storageSvc.UploadBytes(ctx, s3Collection, schematicID, filename, out.Bytes(), "image/webp"); err != nil {
			slog.Error("youtube thumbnail recovery: failed to upload to storage",
				"schematic_id", schematicID, "error", err)
			return
		}

		schem.FeaturedImage = filename
		if err := appStore.Schematics.Update(ctx, schem); err != nil {
			slog.Error("youtube thumbnail recovery: failed to update schematic",
				"schematic_id", schematicID, "error", err)
			return
		}

		// Clear the schematic cache so the next page load picks up the new image
		cacheService.Delete(cache.SchematicKey(schematicID))

		slog.Info("youtube thumbnail recovery: successfully recovered featured image",
			"schematic_id", schematicID, "video_id", videoID, "filename", filename)
	}()
}
