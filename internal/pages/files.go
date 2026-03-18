package pages

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"createmod/internal/server"
	"createmod/internal/storage"

	"github.com/disintegration/imaging"
	"github.com/sunshineplan/imgconv"
)

const (
	maxThumbWidth  = 1920
	maxThumbHeight = 1080
)

// FileServingHandler serves files from S3 with optional ?thumb=WxH image resizing.
// URL pattern: GET /api/files/{collection}/{recordID}/{filename}
func FileServingHandler(storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if storageSvc == nil {
			return e.String(http.StatusServiceUnavailable, "file storage not configured")
		}

		collection := e.Request.PathValue("collection")
		recordID := e.Request.PathValue("recordID")
		filename := e.Request.PathValue("filename")

		if collection == "" || recordID == "" || filename == "" {
			return e.String(http.StatusBadRequest, "missing path parameters")
		}

		// Map collection name to legacy PB collection ID for S3 key lookup
		s3Collection := storage.CollectionPrefix(collection)

		// Parse optional thumb parameter
		thumbParam := e.Request.URL.Query().Get("thumb")
		thumbW, thumbH := parseThumbnailDimensions(thumbParam)

		// Compute ETag from path components (files are immutable by content hash)
		etagInput := collection + "/" + recordID + "/" + filename
		if thumbW > 0 || thumbH > 0 {
			etagInput += fmt.Sprintf("?thumb=%dx%d", thumbW, thumbH)
		}
		h := sha256.Sum256([]byte(etagInput))
		etag := `"` + hex.EncodeToString(h[:8]) + `"`
		e.Response.Header().Set("ETag", etag)

		// Check If-None-Match for conditional requests
		if match := e.Request.Header.Get("If-None-Match"); match != "" && match == etag {
			e.Response.WriteHeader(http.StatusNotModified)
			return nil
		}

		// Set long-lived cache headers (files are immutable by content hash)
		e.Response.Header().Set("Cache-Control", "public, max-age=31536000")

		contentType := detectContentType(filename)

		// No thumbnail requested — stream original file
		if thumbW == 0 && thumbH == 0 {
			reader, err := storageSvc.Download(e.Request.Context(), s3Collection, recordID, filename)
			if err != nil {
				return e.String(http.StatusNotFound, "file not found")
			}
			defer reader.Close()

			e.Response.Header().Set("Content-Type", contentType)
			return e.Stream(http.StatusOK, contentType, reader)
		}

		// Thumbnail requested — check cache, generate if missing
		thumbKey := fmt.Sprintf("_thumbs/%s/%s/%dx%d_%s.webp", s3Collection, recordID, thumbW, thumbH, filename)

		// Try cached thumbnail first
		ctx := e.Request.Context()
		if exists, _ := storageSvc.ExistsRaw(ctx, thumbKey); exists {
			reader, err := storageSvc.DownloadRaw(ctx, thumbKey)
			if err == nil {
				defer reader.Close()
				thumbContentType := detectContentType(filename)
				if isImageFile(filename) {
					thumbContentType = "image/webp"
				}
				e.Response.Header().Set("Content-Type", thumbContentType)
				return e.Stream(http.StatusOK, thumbContentType, reader)
			}
		}

		// Download original for resizing
		reader, err := storageSvc.Download(ctx, s3Collection, recordID, filename)
		if err != nil {
			return e.String(http.StatusNotFound, "file not found")
		}
		defer reader.Close()

		// If not an image file, serve original
		if !isImageFile(filename) {
			e.Response.Header().Set("Content-Type", contentType)
			return e.Stream(http.StatusOK, contentType, reader)
		}

		// Read original image
		originalData, err := io.ReadAll(reader)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to read file")
		}

		// Decode image
		srcImage, _, err := image.Decode(bytes.NewReader(originalData))
		if err != nil {
			// If decoding fails (e.g., unsupported format), serve original
			e.Response.Header().Set("Content-Type", contentType)
			return e.Blob(http.StatusOK, contentType, originalData)
		}

		// Resize
		resized := resizeImage(srcImage, thumbW, thumbH)

		// Encode to WebP
		var thumbBuf bytes.Buffer
		thumbContentType := "image/webp"

		bw := bufio.NewWriter(&thumbBuf)
		if err := imgconv.Write(bw, resized, &imgconv.FormatOption{
			Format:       imgconv.WEBP,
			EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)},
		}); err != nil {
			e.Response.Header().Set("Content-Type", contentType)
			return e.Blob(http.StatusOK, contentType, originalData)
		}
		_ = bw.Flush()

		thumbData := thumbBuf.Bytes()

		// Upload thumbnail to S3 cache in background
		go func() {
			_ = storageSvc.UploadRawBytes(ctx, thumbKey, thumbData, thumbContentType)
		}()

		// Serve the thumbnail
		e.Response.Header().Set("Content-Type", thumbContentType)
		return e.Blob(http.StatusOK, thumbContentType, thumbData)
	}
}

// parseThumbnailDimensions parses "WxH" format. Returns (0,0) for invalid input.
// Supports "640x360", "400x0" (auto height), "0x150" (auto width).
func parseThumbnailDimensions(s string) (int, int) {
	if s == "" {
		return 0, 0
	}

	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}

	w, errW := strconv.Atoi(parts[0])
	h, errH := strconv.Atoi(parts[1])

	if errW != nil || errH != nil {
		return 0, 0
	}

	if w < 0 || h < 0 {
		return 0, 0
	}

	// Cap dimensions
	if w > maxThumbWidth {
		w = maxThumbWidth
	}
	if h > maxThumbHeight {
		h = maxThumbHeight
	}

	return w, h
}

// resizeImage resizes an image to fit within the given dimensions while preserving aspect ratio.
func resizeImage(src image.Image, w, h int) image.Image {
	if w == 0 && h == 0 {
		return src
	}
	return imaging.Fit(src, w, h, imaging.Lanczos)
}

// detectContentType returns the MIME type for a filename based on extension.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Handle known types explicitly
	switch ext {
	case ".nbt":
		return "application/octet-stream"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	}

	// Fall back to mime package
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// isImageFile returns true if the filename has a common image extension.
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		return true
	}
	return false
}