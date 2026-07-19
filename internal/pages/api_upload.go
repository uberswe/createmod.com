package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/nbtparser"
	"createmod/internal/ratelimit"
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
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"createmod/internal/server"
)

const (
	maxGalleryImages  = 10
	maxRotationImages = 130
	maxImageSize      = 5 << 20
	// maxUploadMemory bounds how much of a multipart upload is buffered in RAM;
	// the remainder spills to temp files on disk. Individual files are capped at
	// maxImageSize/maxUploadSize, so a small in-memory budget keeps concurrent
	// uploads from accumulating large buffers and pushing the pod toward OOM.
	maxUploadMemory = 32 << 20
)

var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

func processUploadImages(ctx context.Context, r *http.Request, token string, appStore *store.Store, storageSvc *storage.Service) []uploadImageResponse {
	return processFormImages(ctx, r, "images", "gallery", maxGalleryImages, token, appStore, storageSvc)
}

func processUploadRotationImages(ctx context.Context, r *http.Request, token string, appStore *store.Store, storageSvc *storage.Service) []uploadImageResponse {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}
	imageHeaders, ok := r.MultipartForm.File["rotation_images"]
	if !ok || len(imageHeaders) == 0 {
		return nil
	}
	if len(imageHeaders) > maxRotationImages {
		imageHeaders = imageHeaders[:maxRotationImages]
	}

	type result struct {
		index int
		resp  uploadImageResponse
	}

	results := make([]result, 0, len(imageHeaders))
	var mu sync.Mutex
	var wg sync.WaitGroup
	// Limit concurrent image decodes: each decode can briefly allocate up to
	// ~maxDecodePixels*4 bytes, so memory (not CPU) is the bottleneck here.
	sem := make(chan struct{}, 3)

	for i, fh := range imageHeaders {
		if fh.Size > maxImageSize {
			continue
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedImageExts[ext] {
			continue
		}
		f, openErr := fh.Open()
		if openErr != nil {
			continue
		}
		data, readErr := io.ReadAll(f)
		_ = f.Close()
		if readErr != nil {
			continue
		}

		idx := i
		rawData := data
		rawFilename := fmt.Sprintf("rot_%03d_%s", i, sanitizeFilename(fh.Filename))

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			converted, filename, contentType, convErr := convertToWebP(rawData, rawFilename)
			if convErr != nil {
				slog.Warn("api upload: skipping oversized rotation image", "error", convErr, "token", token[:8], "filename", rawFilename)
				return
			}
			s3Key := s3CollectionTempUploads + "/" + token + "/" + filename
			if storageSvc != nil {
				if uploadErr := storageSvc.UploadRawBytes(ctx, s3Key, converted, contentType); uploadErr != nil {
					slog.Error("api upload: failed to upload rotation image to S3", "error", uploadErr, "token", token[:8], "filename", filename)
					return
				}
			}

			img := &store.TempUploadImage{
				Token:     token,
				Filename:  filename,
				Size:      int64(len(converted)),
				S3Key:     s3Key,
				SortOrder: idx,
				Category:  "rotation",
			}
			if err := appStore.TempUploadImages.Create(ctx, img); err != nil {
				slog.Error("api upload: failed to create temp upload image record", "error", err, "token", token[:8])
				return
			}

			mu.Lock()
			results = append(results, result{
				index: idx,
				resp: uploadImageResponse{
					Filename: filename,
					URL:      "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(filename),
				},
			})
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].index < results[j].index })
	images := make([]uploadImageResponse, len(results))
	for i, r := range results {
		images[i] = r.resp
	}
	return images
}

func processFormImages(ctx context.Context, r *http.Request, formField, category string, maxImages int, token string, appStore *store.Store, storageSvc *storage.Service) []uploadImageResponse {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}
	imageHeaders, ok := r.MultipartForm.File[formField]
	if !ok || len(imageHeaders) == 0 {
		return nil
	}

	var images []uploadImageResponse
	for i, fh := range imageHeaders {
		if i >= maxImages {
			break
		}
		if fh.Size > maxImageSize {
			continue
		}
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedImageExts[ext] {
			continue
		}
		f, openErr := fh.Open()
		if openErr != nil {
			continue
		}
		data, readErr := io.ReadAll(f)
		_ = f.Close()
		if readErr != nil {
			continue
		}

		data, filename, contentType, convErr := convertToWebP(data, sanitizeFilename(fh.Filename))
		if convErr != nil {
			slog.Warn("api upload: skipping oversized image", "error", convErr, "token", token[:8], "filename", fh.Filename)
			continue
		}
		s3Key := s3CollectionTempUploads + "/" + token + "/" + filename
		if storageSvc != nil {
			if uploadErr := storageSvc.UploadRawBytes(ctx, s3Key, data, contentType); uploadErr != nil {
				slog.Error("api upload: failed to upload image to S3", "error", uploadErr, "token", token[:8], "filename", filename)
				continue
			}
		}

		img := &store.TempUploadImage{
			Token:     token,
			Filename:  filename,
			Size:      int64(len(data)),
			S3Key:     s3Key,
			SortOrder: i,
			Category:  category,
		}
		if err := appStore.TempUploadImages.Create(ctx, img); err != nil {
			slog.Error("api upload: failed to create temp upload image record", "error", err, "token", token[:8])
			continue
		}

		images = append(images, uploadImageResponse{
			Filename: filename,
			URL:      "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(filename),
		})
	}
	return images
}

// APIUploadHandler serves POST /api/schematics/upload as a JSON API for uploading schematics.
// Accepts either API key or HMAC authentication. When authenticated via HMAC,
// the upload is anonymous (empty UploadedBy) and can be claimed via /u/{token}/claim.
// Accepts multipart/form-data with an .nbt file.
// The upload goes through the same pipeline as web uploads -- returns a preview token, not a published schematic.
// Uses PostgreSQL store for metadata and direct S3 for file storage.
func APIUploadHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "POST /api/schematics/upload"

		key, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, cacheService)
		if err != nil {
			return nil
		}
		if isHMAC {
			// Rate limit HMAC uploads by IP: 10/min (same as anonymous uploads)
			if ok, retry := searchRateLimitAllow(rl, e.RealIP(), 10); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
		} else {
			defer func() { recordAPIKeyUsageStore(appStore, key.ID, endpoint) }()
			if ok, retry := rateLimitAllow(rl, key.ID, effectiveRateLimit(key, defaultAPIRateLimitPerMinute)); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
		}

		_ = e.Request.ParseMultipartForm(maxUploadMemory)

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
		if header == nil || header.Filename == "" || !isUploadableSchematicName(header.Filename) {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid file type: expected " + UploadAcceptAttr})
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

		// Convert non-.nbt formats to Create/vanilla structure NBT
		uploadFilename := header.Filename
		var convErr error
		data, uploadFilename, _, _, convErr = normalizeUploadToNBT(header.Filename, data)
		if convErr != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": convErr.Error()})
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

		// Duplicate detection happens at the publish step (make-public),
		// not here: temp uploads are private.

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

		// Sanitize the filename to be ASCII-safe for URLs and S3 keys
		safeFilename := sanitizeFilename(uploadFilename)

		// Upload NBT file to S3
		nbtS3Key := s3CollectionTempUploads + "/" + token + "/" + safeFilename
		if storageSvc != nil {
			if err := storageSvc.UploadRawBytes(e.Request.Context(), nbtS3Key, data, "application/octet-stream"); err != nil {
				slog.Error("failed to upload NBT to S3 (API)", "error", err, "token", token[:8])
				return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
			}
		}

		// Persist to PostgreSQL store
		// HMAC-authenticated uploads are anonymous (claimable via /u/{token}/claim)
		uploadedBy := authenticatedUserID(e)
		if isHMAC {
			uploadedBy = ""
		}
		tempUpload := &store.TempUpload{
			Token:         token,
			UploadedBy:    uploadedBy,
			Filename:      safeFilename,
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
			slog.Error("failed to persist temp upload (API)", "error", err, "token", token[:8])
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Process optional image uploads
		uploadedImages := processUploadImages(e.Request.Context(), e.Request, token, appStore, storageSvc)
		uploadedRotation := processUploadRotationImages(e.Request.Context(), e.Request, token, appStore, storageSvc)

		// Build response
		resp := uploadNBTResponse{
			Token:          token,
			URL:            "/u/" + token,
			Checksum:       checksum,
			Filename:       safeFilename,
			Size:           n,
			BlockCount:     blockCount,
			Materials:      parsedMaterials,
			Mods:           mods,
			Images:         uploadedImages,
			RotationImages: uploadedRotation,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ
		resp.FileURL = "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(safeFilename)
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
func APIUploadAnonymousHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		// Rate limit by IP instead of API key
		clientIP := e.RealIP()
		if ok, retry := rateLimitAllow(rl, "anon:"+clientIP, 10); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadMemory)

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
		if header == nil || header.Filename == "" || !isUploadableSchematicName(header.Filename) {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid file type: expected " + UploadAcceptAttr})
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

		// Convert non-.nbt formats to Create/vanilla structure NBT
		uploadFilename := header.Filename
		var convErr error
		data, uploadFilename, _, _, convErr = normalizeUploadToNBT(header.Filename, data)
		if convErr != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": convErr.Error()})
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

		// Duplicate detection happens at the publish step (make-public),
		// not here: temp uploads are private.

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

		// Sanitize the filename to be ASCII-safe for URLs and S3 keys
		safeFilename := sanitizeFilename(uploadFilename)

		// Upload NBT file to S3
		nbtS3Key := s3CollectionTempUploads + "/" + token + "/" + safeFilename
		if storageSvc != nil {
			if err := storageSvc.UploadRawBytes(e.Request.Context(), nbtS3Key, data, "application/octet-stream"); err != nil {
				slog.Error("failed to upload NBT to S3 (anonymous API)", "error", err, "token", token[:8])
				return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
			}
		}

		// Persist to PostgreSQL store — UploadedBy is empty (unclaimed)
		tempUpload := &store.TempUpload{
			Token:         token,
			UploadedBy:    "",
			Filename:      safeFilename,
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
			slog.Error("failed to persist temp upload (anonymous API)", "error", err, "token", token[:8])
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Process optional image uploads
		uploadedImages := processUploadImages(e.Request.Context(), e.Request, token, appStore, storageSvc)
		uploadedRotation := processUploadRotationImages(e.Request.Context(), e.Request, token, appStore, storageSvc)

		// Build response
		resp := uploadNBTResponse{
			Token:          token,
			URL:            "/u/" + token,
			Checksum:       checksum,
			Filename:       safeFilename,
			Size:           n,
			BlockCount:     blockCount,
			Materials:      parsedMaterials,
			Mods:           mods,
			Images:         uploadedImages,
			RotationImages: uploadedRotation,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ
		resp.FileURL = "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(safeFilename)
		if resp.Materials == nil {
			resp.Materials = []nbtparser.Material{}
		}
		if resp.Mods == nil {
			resp.Mods = []string{}
		}

		return writeJSON(e, http.StatusOK, resp)
	}
}
