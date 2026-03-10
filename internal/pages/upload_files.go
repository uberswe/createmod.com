package pages

import (
	"createmod/internal/nbtparser"
	"createmod/internal/storage"
	"createmod/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"createmod/internal/server"
)

// tempUploadFile represents an additional NBT file attached to a temp upload.
type tempUploadFile struct {
	ID          string
	Token       string
	Filename    string
	Description string
	Size        int64
	Checksum    string
	BlockCount  int
	DimX        int
	DimY        int
	DimZ        int
	Mods        []string
	Materials   []nbtparser.Material
	NbtS3Key    string
}

// addFileResponse is the JSON response for a successful additional file upload.
type addFileResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	BlockCount  int    `json:"block_count"`
	DimX        int    `json:"dim_x"`
	DimY        int    `json:"dim_y"`
	DimZ        int    `json:"dim_z"`
	FileURL     string `json:"file_url,omitempty"`
}

// UploadAddFileHandler accepts a POST with an additional .nbt file and description
// for a given temp upload token. Requires auth + ownership.
// Uses PostgreSQL store for metadata and direct S3 for file storage.
func UploadAddFileHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isAuthenticated(e) {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing token"})
		}

		// Verify ownership: the parent temp upload must belong to this user
		parentEntry, err := appStore.TempUploads.GetByToken(e.Request.Context(), token)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
		}
		if parentEntry.UploadedBy == "" || parentEntry.UploadedBy != authenticatedUserID(e) {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "you do not own this upload"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20)

		file, header, err := e.Request.FormFile("nbt")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'nbt')"})
		}
		if file != nil {
			defer file.Close()
		}

		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		if header.Size > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			return e.JSON(http.StatusBadRequest, map[string]string{"error": msg})
		}

		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection within the same token
		existing, _ := appStore.TempUploadFiles.ListByToken(e.Request.Context(), token)
		for _, ef := range existing {
			if ef.Checksum == checksum {
				return e.JSON(http.StatusConflict, map[string]string{"error": "this file has already been added to this upload"})
			}
		}

		// Parse NBT stats
		blockCount, _, _ := nbtparser.ExtractStats(data)
		parsedMaterials, _ := nbtparser.ExtractMaterials(data)
		dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)

		// Extract mod namespaces
		modSet := make(map[string]struct{})
		for _, m := range parsedMaterials {
			parts := strings.SplitN(m.BlockID, ":", 2)
			if len(parts) == 2 && parts[0] != "minecraft" && parts[0] != "" {
				modSet[parts[0]] = struct{}{}
			}
		}
		mods := make([]string, 0, len(modSet))
		for mod := range modSet {
			mods = append(mods, mod)
		}

		description := strings.TrimSpace(e.Request.FormValue("description"))

		// Marshal materials and mods to JSON
		materialsJSON, _ := json.Marshal(parsedMaterials)
		modsJSON, _ := json.Marshal(mods)

		// Create the store record (to get the ID for S3 key)
		tempFile := &store.TempUploadFile{
			Token:       token,
			Filename:    header.Filename,
			Description: description,
			Size:        n,
			Checksum:    checksum,
			BlockCount:  blockCount,
			DimX:        dimX,
			DimY:        dimY,
			DimZ:        dimZ,
			Mods:        modsJSON,
			Materials:   materialsJSON,
		}

		// Upload NBT file to S3 using a temporary key based on checksum
		nbtS3Key := s3CollectionTempUploadFiles + "/" + token + "/" + header.Filename
		if storageSvc != nil {
			if err := storageSvc.UploadRawBytes(e.Request.Context(), nbtS3Key, data, "application/octet-stream"); err != nil {
				slog.Error("failed to upload additional NBT to S3", "error", err, "token", token)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
			}
		}
		tempFile.NbtS3Key = nbtS3Key

		if err := appStore.TempUploadFiles.Create(e.Request.Context(), tempFile); err != nil {
			slog.Error("failed to persist temp upload file", "error", err, "token", token)
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file record"})
		}

		var fileURL string
		if nbtS3Key != "" {
			parts := strings.SplitN(nbtS3Key, "/", 3)
			if len(parts) == 3 {
				fileURL = "/api/files/" + parts[0] + "/" + parts[1] + "/" + url.PathEscape(parts[2])
			} else {
				fileURL = "/api/files/" + nbtS3Key
			}
		}

		return e.JSON(http.StatusOK, addFileResponse{
			ID:          tempFile.ID,
			Filename:    header.Filename,
			Description: description,
			Size:        n,
			BlockCount:  blockCount,
			DimX:        dimX,
			DimY:        dimY,
			DimZ:        dimZ,
			FileURL:     fileURL,
		})
	}
}

// UploadDeleteFileHandler deletes an additional file from a temp upload.
// Requires auth + ownership.
// Uses PostgreSQL store for metadata and direct S3 for file deletion.
func UploadDeleteFileHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if !isAuthenticated(e) {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		token := e.Request.PathValue("token")
		fileId := e.Request.PathValue("fileId")
		if token == "" || fileId == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing token or file ID"})
		}

		// Verify ownership
		parentEntry, err := appStore.TempUploads.GetByToken(e.Request.Context(), token)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
		}
		if parentEntry.UploadedBy == "" || parentEntry.UploadedBy != authenticatedUserID(e) {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "you do not own this upload"})
		}

		// Find the file record
		fileRec, err := appStore.TempUploadFiles.GetByID(e.Request.Context(), fileId)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		}

		// Verify the file belongs to this token
		if fileRec.Token != token {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "file does not belong to this upload"})
		}

		// Delete from S3
		if fileRec.NbtS3Key != "" && storageSvc != nil {
			if err := storageSvc.DeleteRaw(e.Request.Context(), fileRec.NbtS3Key); err != nil {
				slog.Error("failed to delete file from S3", "error", err, "key", fileRec.NbtS3Key)
			}
		}

		// Delete from store
		if err := appStore.TempUploadFiles.Delete(e.Request.Context(), fileId); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete file"})
		}

		return e.NoContent(http.StatusNoContent)
	}
}

// UploadFileDownloadHandler serves an additional NBT file for download.
// Public access (no auth required), matching primary file behavior.
// Uses PostgreSQL store for metadata and direct S3 for file access.
func UploadFileDownloadHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		fileId := e.Request.PathValue("fileId")
		if token == "" || fileId == "" {
			return e.String(http.StatusBadRequest, "missing token or file ID")
		}

		fileRec, err := appStore.TempUploadFiles.GetByID(e.Request.Context(), fileId)
		if err != nil {
			return e.String(http.StatusNotFound, "file not found")
		}

		if fileRec.Token != token {
			return e.String(http.StatusNotFound, "file not found")
		}

		if fileRec.NbtS3Key == "" {
			return e.String(http.StatusNotFound, "file not available")
		}

		return streamFromS3(e, storageSvc, fileRec.NbtS3Key, fileRec.Filename)
	}
}
