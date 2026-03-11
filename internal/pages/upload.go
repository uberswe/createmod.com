package pages

import (
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
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
	"strconv"
	"strings"
	"time"

	"createmod/internal/server"
)

const uploadPendingTemplate = "./template/upload_pending.html"

// maxUploadSize is the maximum allowed NBT file size (10 MB).
const maxUploadSize = 10 * 1024 * 1024

// s3CollectionTempUploads is the S3 prefix for temp upload files.
var s3CollectionTempUploads = storage.CollectionPrefix("temp_uploads")

// s3CollectionTempUploadFiles is the S3 prefix for additional temp upload files.
var s3CollectionTempUploadFiles = storage.CollectionPrefix("temp_upload_files")

const uploadTemplate = "./template/upload.html"
const uploadStepsTemplate = "./template/include/upload_steps.html"

var uploadTemplates = append([]string{
	uploadTemplate,
	uploadStepsTemplate,
}, commonTemplates...)

var uploadPendingTemplates = append([]string{
	uploadPendingTemplate,
}, commonTemplates...)

const uploadPreviewTemplate = "./template/upload_preview.html"

var uploadPreviewTemplates = append([]string{
	uploadPreviewTemplate,
	uploadStepsTemplate,
}, commonTemplates...)

type UploadData struct {
	DefaultData
	UploadStep        int
	PrivateSchematics []store.TempUpload
	PrivatePage       int
	PrivateHasPrev    bool
	PrivateHasNext    bool
	PrivatePrevURL    string
	PrivateNextURL    string
}

type UploadPublishData struct {
	DefaultData
	UploadStep        int
	Token             string
	Filename          string
	Size              int64
	BlockCount        int
	DimX, DimY, DimZ  int
	MinecraftVersions []models.MinecraftVersion
	CreatemodVersions []models.CreatemodVersion
	Tags              []models.SchematicTag
	AdditionalFiles   []tempUploadFile // extra NBT files (variations/sets)
}

type UploadPreviewData struct {
	DefaultData
	UploadStep       int
	Token            string
	Filename         string
	Size             int64
	Checksum         string
	UploadedAt       time.Time
	ParsedSummary    string
	BlockCount       int
	Materials        []string
	ParsedMaterials  []nbtparser.Material
	DimX, DimY, DimZ int
	Mods             []string
	FileURL          string           // path to the NBT file in S3 storage
	IsOwner          bool             // true if current user uploaded this
	AdditionalFiles  []tempUploadFile // extra NBT files (variations/sets)
}

func UploadHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := UploadData{}
		d.Populate(e)
		d.UploadStep = 1
		d.Title = i18n.T(d.Language, "Upload A Schematic")
		d.Description = i18n.T(d.Language, "page.upload.description")
		d.Slug = "/upload"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Load private schematics for authenticated users
		if isAuthenticated(e) {
			userID := authenticatedUserID(e)
			if userID != "" {
				const pageSize = 10
				page := 1
				if p := e.Request.URL.Query().Get("p"); p != "" {
					if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
						page = pv
					}
				}
				offset := (page - 1) * pageSize
				uploads, err := appStore.TempUploads.ListByUser(e.Request.Context(), userID, pageSize+1, offset)
				if err == nil && len(uploads) > 0 {
					d.PrivatePage = page
					d.PrivateHasPrev = page > 1
					if len(uploads) > pageSize {
						d.PrivateHasNext = true
						uploads = uploads[:pageSize]
					}
					d.PrivateSchematics = uploads
					if d.PrivateHasPrev {
						d.PrivatePrevURL = fmt.Sprintf("/upload?p=%d", page-1)
					}
					if d.PrivateHasNext {
						d.PrivateNextURL = fmt.Sprintf("/upload?p=%d", page+1)
					}
				}
			}
		}

		html, err := registry.LoadFiles(uploadTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadPendingHandler renders a simple moderation pending confirmation page.
func UploadPendingHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := DefaultData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Upload Pending Moderation")
		d.Description = i18n.T(d.Language, "page.upload_pending.description")
		d.Slug = "/upload/pending"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		html, err := registry.LoadFiles(uploadPendingTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadMakePublicHandler accepts POSTs to publish a previously uploaded temp schematic.
// Validates the token exists in PostgreSQL and redirects to the moderation pending page.
func UploadMakePublicHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}
		// Verify the token exists in store
		if _, err := appStore.TempUploads.GetByToken(e.Request.Context(), token); err != nil {
			return e.String(http.StatusNotFound, "invalid or expired token")
		}

		// Parse the multipart form (up to 10 MB in memory; the rest spills to
		// temp files).  Using ParseMultipartForm instead of ParseForm ensures
		// that the request body is fully consumed even when the client sends
		// file fields (featured_image, gallery).  Leaving the body unread
		// causes Go's HTTP server to drain or reset the connection, which
		// produces 502 errors through reverse proxies like Cloudflare.
		if err := e.Request.ParseMultipartForm(100 << 20); err == nil {
			val := strings.TrimSpace(e.Request.FormValue("scheduled_at"))
			if val != "" {
				var when time.Time
				var perr error
				when, perr = time.Parse(time.RFC3339, val)
				if perr != nil {
					const layout = "2006-01-02T15:04"
					if t2, err2 := time.ParseInLocation(layout, val, time.Local); err2 == nil {
						when = t2
						perr = nil
					}
				}
				if perr == nil && !when.IsZero() {
					utc := when.UTC()
					key := "upload:schedule:" + token
					cacheService.SetWithTTL(key, utc.Format(time.RFC3339), 24*time.Hour)
				}
			}

			// Resolve user-suggested categories and tags, cache IDs for later schematic creation.
			ctx := e.Request.Context()
			if rawCats := e.Request.Form["categories"]; len(rawCats) > 0 {
				catIDs := resolveCategoryIDs(ctx, appStore, rawCats)
				if len(catIDs) > 0 {
					cacheService.SetWithTTL("upload:categories:"+token, catIDs, 24*time.Hour)
				}
			}
			if rawTags := e.Request.Form["tags"]; len(rawTags) > 0 {
				tagIDs := resolveTagIDs(ctx, appStore, rawTags)
				if len(tagIDs) > 0 {
					cacheService.SetWithTTL("upload:tags:"+token, tagIDs, 24*time.Hour)
				}
			}

			// Cache paid / external_url for later schematic creation
			if e.Request.FormValue("paid") == "true" {
				cacheService.SetWithTTL("upload:paid:"+token, true, 24*time.Hour)
				if eu := strings.TrimSpace(e.Request.FormValue("external_url")); eu != "" {
					cacheService.SetWithTTL("upload:external_url:"+token, eu, 24*time.Hour)
				}
			}
		}

		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/upload/pending"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/upload/pending"))
	}
}

// UploadPreviewHandler serves a minimal private preview page for a given token.
func UploadPreviewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		entry, err := appStore.TempUploads.GetByToken(e.Request.Context(), token)
		if err != nil {
			return e.String(http.StatusNotFound, "invalid or expired token")
		}

		// Determine ownership
		isOwner := isAuthenticated(e) && entry.UploadedBy != "" && authenticatedUserID(e) == entry.UploadedBy

		// Build file URL from S3 key — encode the filename component for safe URLs
		var fileURL string
		if entry.NbtS3Key != "" {
			// S3 key format: {collection}/{recordID}/{filename}
			// Encode only the filename part (last path segment)
			parts := strings.SplitN(entry.NbtS3Key, "/", 3)
			if len(parts) == 3 {
				fileURL = "/api/files/" + parts[0] + "/" + parts[1] + "/" + url.PathEscape(parts[2])
			} else {
				fileURL = "/api/files/" + entry.NbtS3Key
			}
		}

		// Parse materials/mods from JSON
		var parsedMaterials []nbtparser.Material
		if len(entry.Materials) > 0 {
			_ = json.Unmarshal(entry.Materials, &parsedMaterials)
		}
		var mods []string
		if len(entry.Mods) > 0 {
			_ = json.Unmarshal(entry.Mods, &mods)
		}

		// Load additional files from store
		storeFiles, _ := appStore.TempUploadFiles.ListByToken(e.Request.Context(), token)
		additionalFiles := mapStoreTempUploadFiles(storeFiles)

		d := UploadPreviewData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Schematic Review")
		d.Description = i18n.T(d.Language, "page.upload_review.description")
		d.Slug = "/u/" + token
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.UploadStep = 2
		d.Token = token
		d.Filename = entry.Filename
		d.Size = entry.Size
		d.Checksum = entry.Checksum
		d.UploadedAt = entry.Created
		d.ParsedSummary = entry.ParsedSummary
		d.BlockCount = entry.BlockCount
		d.ParsedMaterials = parsedMaterials
		d.DimX = entry.DimX
		d.DimY = entry.DimY
		d.DimZ = entry.DimZ
		d.Mods = mods
		d.FileURL = fileURL
		d.IsOwner = isOwner
		d.AdditionalFiles = additionalFiles
		h, err := registry.LoadFiles(uploadPreviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, h)
	}
}

// UploadDownloadHandler serves the NBT file for a given token as a download.
// Uses PostgreSQL store for metadata and direct S3 for file access.
func UploadDownloadHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		entry, err := appStore.TempUploads.GetByToken(e.Request.Context(), token)
		if err != nil {
			return e.String(http.StatusNotFound, "invalid or expired token")
		}

		if entry.NbtS3Key == "" {
			return e.String(http.StatusNotFound, "file not available")
		}

		reader, err := storageSvc.DownloadRaw(e.Request.Context(), entry.NbtS3Key)
		if err != nil {
			return e.String(http.StatusNotFound, "file not found in storage")
		}
		defer reader.Close()

		e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeContentDispositionFilename(entry.Filename)+"\"")
		e.Response.Header().Set("Content-Type", "application/octet-stream")
		return e.Stream(http.StatusOK, "application/octet-stream", reader)
	}
}

// uploadNBTResponse is the JSON response for a successful NBT upload.
type uploadNBTResponse struct {
	Token    string `json:"token"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	FileURL  string `json:"file_url,omitempty"`
	Dimensions struct {
		X int `json:"x"`
		Y int `json:"y"`
		Z int `json:"z"`
	} `json:"dimensions"`
	BlockCount int                  `json:"block_count"`
	Materials  []nbtparser.Material `json:"materials"`
	Mods       []string             `json:"mods"`
}

// UploadNBTHandler validates an uploaded .nbt file, parses stats, persists to
// PostgreSQL + S3, and returns a JSON response with token, dimensions, materials, and mods.
func UploadNBTHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20) // slight overhead for multipart framing
		// Attempt to read the file field (common names: "nbt" or "file").
		file, header, err := e.Request.FormFile("nbt")
		if err != nil {
			file, header, err = e.Request.FormFile("file")
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'nbt' or 'file')"})
			}
		}
		if file != nil {
			defer file.Close()
		}
		// Basic filename validation before parsing
		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		// 10MB size limit check
		if header.Size > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}
		// Read the uploaded file fully into memory to compute size and checksum
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}
		// Minimal backend validation
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

		// Duplicate detection -- skipped in dev mode (DEV=true)
		// Only checks against published (moderated) schematics and blacklisted
		// hashes. Temp/private uploads are intentionally not checked so users
		// can re-upload after losing their token or making a mistake.
		isDev := os.Getenv("DEV") == "true"
		if !isDev {
			dupMsg := "This schematic already exists (duplicate upload detected by checksum). It may be blacklisted by the original creator. If you need more help contact us: /contact"

			// Check published schematics via store
			if appStore != nil {
				if existingID, err := appStore.Schematics.GetByChecksum(context.Background(), checksum); err == nil && existingID != "" {
					return e.JSON(http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}

			// Check blacklist hashes
			if appStore != nil {
				if blacklisted, err := appStore.NBTHashes.IsBlacklisted(context.Background(), checksum); err == nil && blacklisted {
					return e.JSON(http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
		}

		// Generate a random token
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate preview token"})
		}
		token := hex.EncodeToString(buf)

		// Parse summary (legacy)
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = fmt.Sprintf("size=%d bytes; nbt=unparsed", n)
		}

		// Extract stats
		blockCount, _, _ := nbtparser.ExtractStats(data)

		// Extract enriched materials
		parsedMaterials, _ := nbtparser.ExtractMaterials(data)

		// Extract dimensions
		dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)

		// Extract mod namespaces from materials
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
				slog.Error("failed to upload NBT to S3", "error", err, "token", token)
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
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
			slog.Error("failed to persist temp upload", "error", err, "token", token)
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Build file URL — encode the filename component for safe use in URLs
		fileURL := "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(header.Filename)

		// Build JSON response
		resp := uploadNBTResponse{
			Token:      token,
			URL:        "/u/" + token,
			Checksum:   checksum,
			Filename:   header.Filename,
			Size:       n,
			FileURL:    fileURL,
			BlockCount: blockCount,
			Materials:  parsedMaterials,
			Mods:       mods,
		}
		resp.Dimensions.X = dimX
		resp.Dimensions.Y = dimY
		resp.Dimensions.Z = dimZ

		if resp.Materials == nil {
			resp.Materials = []nbtparser.Material{}
		}
		if resp.Mods == nil {
			resp.Mods = []string{}
		}

		return e.JSON(http.StatusOK, resp)
	}
}

// mapStoreTempUploadFiles converts store.TempUploadFile slice to the template-facing tempUploadFile slice.
func mapStoreTempUploadFiles(files []store.TempUploadFile) []tempUploadFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]tempUploadFile, len(files))
	for i, f := range files {
		var materials []nbtparser.Material
		if len(f.Materials) > 0 {
			_ = json.Unmarshal(f.Materials, &materials)
		}
		var mods []string
		if len(f.Mods) > 0 {
			_ = json.Unmarshal(f.Mods, &mods)
		}
		result[i] = tempUploadFile{
			ID:          f.ID,
			Token:       f.Token,
			Filename:    f.Filename,
			Description: f.Description,
			Size:        f.Size,
			Checksum:    f.Checksum,
			BlockCount:  f.BlockCount,
			DimX:        f.DimX,
			DimY:        f.DimY,
			DimZ:        f.DimZ,
			Mods:        mods,
			Materials:   materials,
			NbtS3Key:    f.NbtS3Key,
		}
	}
	return result
}

// streamFromS3 is a helper that downloads a file from S3 by raw key and streams it as an attachment.
func streamFromS3(e *server.RequestEvent, storageSvc *storage.Service, s3Key, filename string) error {
	reader, err := storageSvc.DownloadRaw(e.Request.Context(), s3Key)
	if err != nil {
		return e.String(http.StatusNotFound, "file not found in storage")
	}
	defer reader.Close()

	// Buffer enough to detect valid content (minio returns empty reader on missing keys)
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, reader, 1); err != nil {
		return e.String(http.StatusNotFound, "file not found in storage")
	}

	e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeContentDispositionFilename(filename)+"\"")
	e.Response.Header().Set("Content-Type", "application/octet-stream")
	combined := io.MultiReader(&buf, reader)
	return e.Stream(http.StatusOK, "application/octet-stream", combined)
}

func allCreatemodVersionsFromStore(appStore *store.Store) []models.CreatemodVersion {
	versions, err := appStore.VersionLookup.ListCreatemodVersions(context.Background())
	if err != nil {
		return nil
	}
	result := make([]models.CreatemodVersion, len(versions))
	for i, v := range versions {
		result[i] = models.CreatemodVersion{
			ID:      v.ID,
			Version: v.Version,
		}
	}
	return result
}

func allMinecraftVersionsFromStore(appStore *store.Store) []models.MinecraftVersion {
	versions, err := appStore.VersionLookup.ListMinecraftVersions(context.Background())
	if err != nil {
		return nil
	}
	result := make([]models.MinecraftVersion, len(versions))
	for i, v := range versions {
		result[i] = models.MinecraftVersion{
			ID:      v.ID,
			Version: v.Version,
		}
	}
	return result
}
