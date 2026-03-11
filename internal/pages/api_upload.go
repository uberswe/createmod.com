package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/nbtparser"
	"createmod/internal/storage"
	"createmod/internal/store"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"createmod/internal/server"
)

// APIUploadHandler serves POST /api/schematics/upload as a JSON API for uploading schematics.
// Requires API key authentication. Accepts multipart/form-data with an .nbt file.
// The upload goes through the same pipeline as web uploads -- returns a preview token, not a published schematic.
// Uses PostgreSQL store for metadata and direct S3 for file storage.
func APIUploadHandler(cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "POST /api/schematics/upload"

		keyID, err := requireAPIKeyFromStore(appStore, e)
		if err != nil {
			return nil
		}
		defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()

		if ok, retry := rateLimitAllow(cacheService, keyID, 120); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20)

		// Read file from form (field name "file" or "nbt")
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			file, header, err = e.Request.FormFile("nbt")
			if err != nil {
				return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'file' or 'nbt')"})
			}
		}
		if file != nil {
			defer file.Close()
		}

		// Validate filename
		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		if header.Size > maxUploadSize {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Read file data
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Validate NBT
		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": msg})
		}

		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection (skip in dev mode)
		// Only checks against published (moderated) schematics and blacklisted
		// hashes. Temp/private uploads are intentionally not checked so users
		// can re-upload after losing their token or making a mistake.
		isDev := os.Getenv("DEV") == "true"
		if !isDev {
			dupMsg := "This schematic already exists (duplicate upload detected by checksum). It may be blacklisted by the original creator."

			// Check published schematics via store
			if appStore != nil {
				if existingID, err := appStore.Schematics.GetByChecksum(context.Background(), checksum); err == nil && existingID != "" {
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}

			// Check blacklist hashes
			if appStore != nil {
				if blacklisted, err := appStore.NBTHashes.IsBlacklisted(context.Background(), checksum); err == nil && blacklisted {
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
		}

		// Generate token
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to generate preview token"})
		}
		token := hex.EncodeToString(buf)

		// Parse summary
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = fmt.Sprintf("size=%d bytes; nbt=unparsed", n)
		}

		// Extract stats
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

		// Marshal materials and mods to JSON for storage
		materialsJSON, _ := json.Marshal(parsedMaterials)
		modsJSON, _ := json.Marshal(mods)

		// Upload NBT file to S3
		nbtS3Key := s3CollectionTempUploads + "/" + token + "/" + header.Filename
		if storageSvc != nil {
			if err := storageSvc.UploadRawBytes(e.Request.Context(), nbtS3Key, data, "application/octet-stream"); err != nil {
				slog.Error("failed to upload NBT to S3 (API)", "error", err, "token", token)
				return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
			}
		}

		// Persist to PostgreSQL store
		tempUpload := &store.TempUpload{
			Token:         token,
			UploadedBy:    authenticatedUserID(e),
			Filename:      header.Filename,
			Size:          n,
			Checksum:      checksum,
			BlockCount:    blockCount,
			DimX:          dimX,
			DimY:          dimY,
			DimZ:          dimZ,
			Mods:          modsJSON,
			Materials:     materialsJSON,
			NbtS3Key:      nbtS3Key,
			ParsedSummary: parsed,
		}

		if err := appStore.TempUploads.Create(e.Request.Context(), tempUpload); err != nil {
			slog.Error("failed to persist temp upload (API)", "error", err, "token", token)
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Build response
		resp := uploadNBTResponse{
			Token:      token,
			URL:        "/u/" + token,
			Checksum:   checksum,
			Filename:   header.Filename,
			Size:       n,
			BlockCount: blockCount,
			Materials:  parsedMaterials,
			Mods:       mods,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ
		resp.FileURL = "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(header.Filename)
		if resp.Materials == nil {
			resp.Materials = []nbtparser.Material{}
		}
		if resp.Mods == nil {
			resp.Mods = []string{}
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}

// APIUploadAnonymousHandler serves POST /api/schematics/upload-anonymous as an
// unauthenticated JSON API for uploading schematics. No API key is required.
// Rate-limited by client IP (10 uploads/min). The upload is created with an
// empty UploadedBy so it can later be claimed via /u/{token}/claim.
func APIUploadAnonymousHandler(cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Rate limit by IP instead of API key
		clientIP := e.RealIP()
		if ok, retry := rateLimitAllow(cacheService, "anon:"+clientIP, 10); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20)

		// Read file from form (field name "file" or "nbt")
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			file, header, err = e.Request.FormFile("nbt")
			if err != nil {
				return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'file' or 'nbt')"})
			}
		}
		if file != nil {
			defer file.Close()
		}

		// Validate filename
		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		if header.Size > maxUploadSize {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Read file data
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		// Validate NBT
		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": msg})
		}

		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection (skip in dev mode)
		isDev := os.Getenv("DEV") == "true"
		if !isDev {
			dupMsg := "This schematic already exists (duplicate upload detected by checksum). It may be blacklisted by the original creator."

			if appStore != nil {
				if existingID, err := appStore.Schematics.GetByChecksum(context.Background(), checksum); err == nil && existingID != "" {
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}

			if appStore != nil {
				if blacklisted, err := appStore.NBTHashes.IsBlacklisted(context.Background(), checksum); err == nil && blacklisted {
					return writeJSON(e, http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
		}

		// Generate token
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to generate preview token"})
		}
		token := hex.EncodeToString(buf)

		// Parse summary
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = fmt.Sprintf("size=%d bytes; nbt=unparsed", n)
		}

		// Extract stats
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

		// Marshal materials and mods to JSON for storage
		materialsJSON, _ := json.Marshal(parsedMaterials)
		modsJSON, _ := json.Marshal(mods)

		// Upload NBT file to S3
		nbtS3Key := s3CollectionTempUploads + "/" + token + "/" + header.Filename
		if storageSvc != nil {
			if err := storageSvc.UploadRawBytes(e.Request.Context(), nbtS3Key, data, "application/octet-stream"); err != nil {
				slog.Error("failed to upload NBT to S3 (anonymous API)", "error", err, "token", token)
				return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
			}
		}

		// Persist to PostgreSQL store — UploadedBy is empty (unclaimed)
		tempUpload := &store.TempUpload{
			Token:         token,
			UploadedBy:    "",
			Filename:      header.Filename,
			Size:          n,
			Checksum:      checksum,
			BlockCount:    blockCount,
			DimX:          dimX,
			DimY:          dimY,
			DimZ:          dimZ,
			Mods:          modsJSON,
			Materials:     materialsJSON,
			NbtS3Key:      nbtS3Key,
			ParsedSummary: parsed,
		}

		if err := appStore.TempUploads.Create(e.Request.Context(), tempUpload); err != nil {
			slog.Error("failed to persist temp upload (anonymous API)", "error", err, "token", token)
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Build response
		resp := uploadNBTResponse{
			Token:      token,
			URL:        "/u/" + token,
			Checksum:   checksum,
			Filename:   header.Filename,
			Size:       n,
			BlockCount: blockCount,
			Materials:  parsedMaterials,
			Mods:       mods,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ
		resp.FileURL = "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(header.Filename)
		if resp.Materials == nil {
			resp.Materials = []nbtparser.Material{}
		}
		if resp.Mods == nil {
			resp.Mods = []string{}
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}
