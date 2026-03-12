package pages

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"log/slog"

	"createmod/internal/storage"
	"createmod/internal/store"

	"github.com/sunshineplan/imgconv"
	"golang.org/x/image/draw"
)

// generateCollectionCollage builds a 2x2 (or smaller) collage from the first
// four schematics in a collection, uploads it to S3 and updates the DB.
// It is designed to be called as a goroutine (fire-and-forget).
func generateCollectionCollage(storageSvc *storage.Service, appStore *store.Store, collectionID string) {
	if storageSvc == nil {
		return
	}

	ctx := context.Background()

	ids, err := appStore.Collections.GetSchematicIDs(ctx, collectionID)
	if err != nil || len(ids) == 0 {
		return
	}

	// Limit to 4 images
	if len(ids) > 4 {
		ids = ids[:4]
	}

	schematics, err := appStore.Schematics.ListByIDs(ctx, ids)
	if err != nil || len(schematics) == 0 {
		return
	}

	// Collect images in order
	var images []image.Image
	for _, id := range ids {
		for _, s := range schematics {
			if s.ID == id && s.FeaturedImage != "" {
				key := fmt.Sprintf("schematics/%s/%s", s.ID, s.FeaturedImage)
				rc, err := storageSvc.DownloadRaw(ctx, key)
				if err != nil {
					slog.Debug("collage: failed to download image", "key", key, "error", err)
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
				break
			}
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
		// Single image fills entire canvas
		draw.CatmullRom.Scale(dst, dst.Bounds(), images[0], images[0].Bounds(), draw.Over, nil)
	case 2:
		// Two images side by side
		halfW := collageW / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, collageH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, collageH), images[1], images[1].Bounds(), draw.Over, nil)
	case 3:
		// First image on left half, two stacked on right
		halfW := collageW / 2
		halfH := collageH / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, collageH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, halfH), images[1], images[1].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, halfH, collageW, collageH), images[2], images[2].Bounds(), draw.Over, nil)
	default:
		// 2x2 grid
		halfW := collageW / 2
		halfH := collageH / 2
		draw.CatmullRom.Scale(dst, image.Rect(0, 0, halfW, halfH), images[0], images[0].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, 0, collageW, halfH), images[1], images[1].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(0, halfH, halfW, collageH), images[2], images[2].Bounds(), draw.Over, nil)
		draw.CatmullRom.Scale(dst, image.Rect(halfW, halfH, collageW, collageH), images[3], images[3].Bounds(), draw.Over, nil)
	}

	var out bytes.Buffer
	bw := bufio.NewWriter(&out)
	if err := imgconv.Write(bw, dst, &imgconv.FormatOption{Format: imgconv.WEBP, EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)}}); err != nil {
		slog.Error("collage: failed to encode", "error", err)
		return
	}
	_ = bw.Flush()

	imageID, err := generateImageID()
	if err != nil {
		slog.Error("collage: failed to generate image ID", "error", err)
		return
	}
	filename := "collage.webp"
	if err := storageSvc.UploadBytes(ctx, "images", imageID, filename, out.Bytes(), "image/webp"); err != nil {
		slog.Error("collage: failed to upload", "error", err)
		return
	}

	collageURL := fmt.Sprintf("/api/files/images/%s/%s", imageID, filename)
	if err := appStore.Collections.UpdateCollageURL(ctx, collectionID, collageURL); err != nil {
		slog.Error("collage: failed to update DB", "error", err)
	}
}
