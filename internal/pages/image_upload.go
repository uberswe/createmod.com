package pages

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"createmod/internal/server"
	"createmod/internal/storage"
)

const maxImageUploadSize = 5 << 20 // 5MB

// allowedImageTypes maps extensions to MIME types for upload validation.
var allowedImageTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func generateImageID() string {
	return randomHex(8)[:15]
}

// ImageUploadHandler handles POST /api/images/upload for authenticated users.
// Returns JSON {"location": "/api/files/images/{id}/{filename}"} for TinyMCE compatibility.
func ImageUploadHandler(storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if storageSvc == nil {
			return writeJSON(e, http.StatusServiceUnavailable, map[string]string{"error": "file storage not configured"})
		}

		// Require authentication
		if !isAuthenticated(e) {
			return writeJSON(e, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		if err := e.Request.ParseMultipartForm(maxImageUploadSize); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "request too large (max 5MB)"})
		}

		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		}
		defer file.Close()

		if header.Size > maxImageUploadSize {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large (max 5MB)"})
		}

		// Validate file extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if _, ok := allowedImageTypes[ext]; !ok {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "unsupported file type (allowed: png, jpg, jpeg, webp, gif)"})
		}

		// Read file data
		data, err := io.ReadAll(file)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
		}

		// Convert to WebP, stripping metadata. Rejects decompression bombs and animated GIFs.
		data, filename, contentType, convErr := convertToWebP(data, sanitizeFilename(filepath.Base(header.Filename)))
		if errors.Is(convErr, errImageTooLarge) {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "image resolution too large"})
		}
		if errors.Is(convErr, errAnimatedGIF) {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "animated GIFs are not allowed"})
		}

		// Generate unique ID
		imageID := generateImageID()

		// Upload to S3
		if err := storageSvc.UploadBytes(e.Request.Context(), "images", imageID, filename, data, contentType); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to upload image"})
		}

		// Return location for TinyMCE
		location := "/api/files/images/" + imageID + "/" + url.PathEscape(filename)
		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		return json.NewEncoder(e.Response).Encode(map[string]string{"location": location})
	}
}
