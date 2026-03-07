package pages

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"createmod/internal/server"
	"createmod/internal/storage"

	"github.com/sunshineplan/imgconv"
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

// generateImageID returns a random 15-character hex string for use as a record ID.
func generateImageID() (string, error) {
	buf := make([]byte, 8) // 16 hex chars, we trim to 15
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:15], nil
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
		contentType, ok := allowedImageTypes[ext]
		if !ok {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "unsupported file type (allowed: png, jpg, jpeg, webp, gif)"})
		}

		// Read file data
		data, err := io.ReadAll(file)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
		}

		// Convert to WebP (skip GIF and already-WebP files)
		filename := filepath.Base(header.Filename)
		if ext != ".gif" && ext != ".webp" {
			img, decErr := imgconv.Decode(bytes.NewReader(data))
			if decErr == nil {
				var out bytes.Buffer
				bw := bufio.NewWriter(&out)
				if encErr := imgconv.Write(bw, img, &imgconv.FormatOption{
					Format:       imgconv.WEBP,
					EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)},
				}); encErr == nil {
					_ = bw.Flush()
					data = out.Bytes()
					contentType = "image/webp"
					// Replace extension with .webp
					baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
					filename = baseName + ".webp"
				}
			}
		}

		// Generate unique ID
		imageID, err := generateImageID()
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to generate ID"})
		}

		// Upload to S3
		if err := storageSvc.UploadBytes(e.Request.Context(), "images", imageID, filename, data, contentType); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to upload image"})
		}

		// Return location for TinyMCE
		location := "/api/files/images/" + imageID + "/" + filename
		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		return json.NewEncoder(e.Response).Encode(map[string]string{"location": location})
	}
}
