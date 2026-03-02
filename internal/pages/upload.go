package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/nbtparser"
	"createmod/internal/store"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const uploadPendingTemplate = "./template/upload_pending.html"

// maxUploadSize is the maximum allowed NBT file size (10 MB).
const maxUploadSize = 10 * 1024 * 1024

type tempUpload struct {
	Filename        string
	Size            int64
	Checksum        string
	UploadedAt      time.Time
	ParsedSummary   string
	BlockCount      int
	Materials       []string             // legacy state:id=count format
	ParsedMaterials []nbtparser.Material // enriched materials with block names
	DimX, DimY, DimZ int                // dimensions
	Mods            []string             // detected mod namespaces
	NBTData         []byte               // raw NBT file bytes (for PB file upload)
	PBRecordID      string               // PocketBase record ID after persist
	NBTStoredName   string               // filename as stored in PB (for file URL)
	UploadedBy      string               // user ID of uploader (empty if anonymous)
}

// note: For durable tokens, we attempt to persist to PocketBase collection
// "temp_uploads" when available. The schema should include fields:
// - token (text, unique)
// - filename (text)
// - size (number)
// - checksum (text)
// - parsed_summary (text)
// This keeps token links working across restarts. We still maintain an
// in-memory fallback so tests and local setups without the collection work.

var tempUploadStore = struct {
	sync.RWMutex
	m map[string]tempUpload
}{
	m: make(map[string]tempUpload),
}

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
	UploadStep int
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
	FileURL          string           // path to the NBT file in PB storage
	IsOwner          bool             // true if current user uploaded this
	AdditionalFiles  []tempUploadFile // extra NBT files (variations/sets)
}

func UploadHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := UploadData{}
		d.Populate(e)
		d.UploadStep = 1
		d.Title = i18n.T(d.Language, "Upload A Schematic")
		d.Description = i18n.T(d.Language, "page.upload.description")
		d.Slug = "/upload"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(uploadTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadPendingHandler renders a simple moderation pending confirmation page.
func UploadPendingHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := DefaultData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Upload Pending Moderation")
		d.Description = i18n.T(d.Language, "page.upload_pending.description")
		d.Slug = "/upload/pending"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		html, err := registry.LoadFiles(uploadPendingTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadMakePublicHandler accepts POSTs to publish a previously uploaded temp schematic.
// Minimal implementation: validate the token exists (PB or in-memory) and redirect to the
// moderation pending page. Future work will create the schematic record and move the file.
func UploadMakePublicHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}
		// Verify the token exists in PB or in-memory store
		if pbEntry, ok := loadTempUploadPB(app, token); !ok {
			tempUploadStore.RLock()
			_, ok2 := tempUploadStore.m[token]
			tempUploadStore.RUnlock()
			if !ok2 {
				return e.String(http.StatusNotFound, "invalid or expired token")
			}
		} else {
			_ = pbEntry // not used yet; placeholder for future mapping
		}

		// Parse optional scheduled_at from form and cache in UTC for later use
		if err := e.Request.ParseForm(); err == nil {
			val := strings.TrimSpace(e.Request.FormValue("scheduled_at"))
			if val != "" {
				var when time.Time
				var perr error
				// Try RFC3339 first
				when, perr = time.Parse(time.RFC3339, val)
				if perr != nil {
					// Fallback to HTML datetime-local (no timezone)
					// Interpret as local time
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
		}

		// HTMX-aware redirect to avoid partial update mismatch
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/upload/pending"))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/upload/pending"))
	}
}

func allCreatemodVersions(app *pocketbase.PocketBase) []models.CreatemodVersion {
	createmodVersionCollection, err := app.FindCollectionByNameOrId("createmod_versions")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(createmodVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToCreatemodVersions(records)
}

func mapResultToCreatemodVersions(records []*core.Record) []models.CreatemodVersion {
	versions := make([]models.CreatemodVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.CreatemodVersion{
			ID:      r.Id,
			Version: r.GetString("version"),
		})
	}
	return versions
}

func allMinecraftVersions(app *pocketbase.PocketBase) []models.MinecraftVersion {
	minecraftVersionCollection, err := app.FindCollectionByNameOrId("minecraft_versions")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(minecraftVersionCollection.Id, "1=1", "-version", -1, 0)
	if err != nil {
		return nil
	}
	return mapResultToMinecraftVersions(records)
}

func mapResultToMinecraftVersions(records []*core.Record) []models.MinecraftVersion {
	versions := make([]models.MinecraftVersion, 0, len(records))
	for _, r := range records {
		versions = append(versions, models.MinecraftVersion{
			ID:      r.Id,
			Version: r.GetString("version"),
		})
	}
	return versions
}

// UploadPreviewHandler serves a minimal private preview page for a given token.
func UploadPreviewHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}
		// Resolve entry from PB or in-memory store
		var entry tempUpload
		if pbEntry, ok := loadTempUploadPB(app, token); ok {
			entry = pbEntry
		} else {
			tempUploadStore.RLock()
			v, ok := tempUploadStore.m[token]
			tempUploadStore.RUnlock()
			if !ok {
				return e.String(http.StatusNotFound, "invalid or expired token")
			}
			entry = v
		}
		// Determine ownership
		isOwner := false
		if isAuthenticated(e) && entry.UploadedBy != "" && authenticatedUserID(e) == entry.UploadedBy {
			isOwner = true
		}

		// Build file URL
		var fileURL string
		if entry.PBRecordID != "" && entry.NBTStoredName != "" {
			fileURL = "/api/files/temp_uploads/" + entry.PBRecordID + "/" + entry.NBTStoredName
		}

		// Render template with review data
		d := UploadPreviewData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Schematic Review")
		d.Description = i18n.T(d.Language, "page.upload_review.description")
		d.Slug = "/u/" + token
		d.Categories = allCategoriesFromStore(appStore, app, cacheService)
		d.UploadStep = 2
		d.Token = token
		d.Filename = entry.Filename
		d.Size = entry.Size
		d.Checksum = entry.Checksum
		d.UploadedAt = entry.UploadedAt
		d.ParsedSummary = entry.ParsedSummary
		d.BlockCount = entry.BlockCount
		d.Materials = entry.Materials
		d.ParsedMaterials = entry.ParsedMaterials
		d.DimX = entry.DimX
		d.DimY = entry.DimY
		d.DimZ = entry.DimZ
		d.Mods = entry.Mods
		d.FileURL = fileURL
		d.IsOwner = isOwner
		d.AdditionalFiles = loadTempUploadFiles(app, token)
		h, err := registry.LoadFiles(uploadPreviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, h)
	}
}

// UploadDownloadHandler serves the NBT file for a given token as a download.
func UploadDownloadHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		// Resolve entry from PB or in-memory store
		var entry tempUpload
		if pbEntry, ok := loadTempUploadPB(app, token); ok {
			entry = pbEntry
		} else {
			tempUploadStore.RLock()
			v, ok := tempUploadStore.m[token]
			tempUploadStore.RUnlock()
			if !ok {
				return e.String(http.StatusNotFound, "invalid or expired token")
			}
			entry = v
		}

		// If the file is in PB storage, serve it with download headers
		if entry.PBRecordID != "" && entry.NBTStoredName != "" {
			coll, err := app.FindCollectionByNameOrId("temp_uploads")
			if err != nil {
				return e.String(http.StatusInternalServerError, "collection not found")
			}
			fileKey := coll.Id + "/" + entry.PBRecordID + "/" + entry.NBTStoredName
			fsys, err := app.NewFilesystem()
			if err != nil {
				return e.String(http.StatusInternalServerError, "storage error")
			}
			defer fsys.Close()

			blob, err := fsys.GetReader(fileKey)
			if err != nil {
				return e.String(http.StatusNotFound, "file not found in storage")
			}
			defer blob.Close()

			e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+entry.Filename+"\"")
			e.Response.Header().Set("Content-Type", "application/octet-stream")
			return e.Stream(http.StatusOK, "application/octet-stream", blob)
		}

		// Fallback: serve from in-memory NBT data
		if len(entry.NBTData) > 0 {
			e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+entry.Filename+"\"")
			e.Response.Header().Set("Content-Type", "application/octet-stream")
			return e.Blob(http.StatusOK, "application/octet-stream", entry.NBTData)
		}

		return e.String(http.StatusNotFound, "file not available")
	}
}

// uploadNBTResponse is the JSON response for a successful NBT upload.
type uploadNBTResponse struct {
	Token      string               `json:"token"`
	URL        string               `json:"url"`
	Checksum   string               `json:"checksum"`
	Filename   string               `json:"filename"`
	Size       int64                `json:"size"`
	FileURL    string               `json:"file_url,omitempty"`
	Dimensions struct {
		X int `json:"x"`
		Y int `json:"y"`
		Z int `json:"z"`
	} `json:"dimensions"`
	BlockCount int                  `json:"block_count"`
	Materials  []nbtparser.Material `json:"materials"`
	Mods       []string             `json:"mods"`
}

// UploadNBTHandler validates an uploaded .nbt file, parses stats, and returns
// a JSON response with token, dimensions, materials, and detected mods.
func UploadNBTHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		// Duplicate detection — skipped in dev mode (DEV=true)
		isDev := os.Getenv("DEV") == "true"
		if !isDev {
			dupMsg := "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation, otherwise it may be blacklisted by the original creator. If you need more help contact us: /contact"
			if app != nil {
				if coll, err := app.FindCollectionByNameOrId("temp_uploads"); err == nil && coll != nil {
					recs, err := app.FindRecordsByFilter(coll.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
					if err == nil && len(recs) > 0 {
						return e.JSON(http.StatusConflict, map[string]string{"error": dupMsg})
					}
				}
				if schemColl, err := app.FindCollectionByNameOrId("schematics"); err == nil && schemColl != nil {
					recs, err := app.FindRecordsByFilter(schemColl.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
					if err == nil && len(recs) > 0 {
						return e.JSON(http.StatusConflict, map[string]string{"error": dupMsg})
					}
				}
			}

			// Duplicate detection (in-memory temp store)
			tempUploadStore.RLock()
			for _, entry := range tempUploadStore.m {
				if entry.Checksum == checksum {
					tempUploadStore.RUnlock()
					return e.JSON(http.StatusConflict, map[string]string{"error": dupMsg})
				}
			}
			tempUploadStore.RUnlock()
		}

		// Best-effort: persist hash to nbt_hashes
		if app != nil {
			if coll, err := app.FindCollectionByNameOrId("nbt_hashes"); err == nil && coll != nil {
				rec := core.NewRecord(coll)
				rec.Set("checksum", checksum)
				if isAuthenticated(e) {
					rec.Set("uploaded_by", authenticatedUserID(e))
				}
				_ = app.Save(rec)
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
		blockCount, legacyMaterials, _ := nbtparser.ExtractStats(data)

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

		// Store metadata in the in-memory temporary store
		tempUploadStore.Lock()
		tempUploadStore.m[token] = tempUpload{
			Filename:        header.Filename,
			Size:            n,
			Checksum:        checksum,
			UploadedAt:      time.Now(),
			ParsedSummary:   parsed,
			BlockCount:      blockCount,
			Materials:       legacyMaterials,
			ParsedMaterials: parsedMaterials,
			DimX:            dimX,
			DimY:            dimY,
			DimZ:            dimZ,
			Mods:            mods,
			NBTData:         data,
			UploadedBy:      authenticatedUserID(e),
		}
		entry := tempUploadStore.m[token]
		tempUploadStore.Unlock()

		// Best-effort: persist to PocketBase so the token link survives restarts
		recID, storedName := persistTempUploadPB(app, token, entry)

		// Update in-memory entry with PB record info
		if recID != "" {
			tempUploadStore.Lock()
			e2 := tempUploadStore.m[token]
			e2.PBRecordID = recID
			e2.NBTStoredName = storedName
			e2.NBTData = nil // free memory after persisting
			tempUploadStore.m[token] = e2
			tempUploadStore.Unlock()
		}

		// Build file URL if the file was persisted
		var fileURL string
		if recID != "" && storedName != "" {
			fileURL = "/api/files/temp_uploads/" + recID + "/" + storedName
		}

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

// --- Persistent storage helpers (best-effort PocketBase) ---
// persistTempUploadPB attempts to save the temporary upload entry in the
// "temp_uploads" collection. If the collection doesn't exist, it silently
// returns without error so local/dev and tests without migrations keep working.
// Returns (recordID, storedFilename) on success.
func persistTempUploadPB(app *pocketbase.PocketBase, token string, entry tempUpload) (string, string) {
	if app == nil {
		return "", ""
	}
	coll, err := app.FindCollectionByNameOrId("temp_uploads")
	if err != nil || coll == nil {
		return "", ""
	}
	rec := core.NewRecord(coll)
	rec.Set("token", token)
	rec.Set("filename", entry.Filename)
	rec.Set("size", entry.Size)
	rec.Set("checksum", entry.Checksum)
	rec.Set("parsed_summary", entry.ParsedSummary)
	rec.Set("block_count", entry.BlockCount)
	rec.Set("dim_x", entry.DimX)
	rec.Set("dim_y", entry.DimY)
	rec.Set("dim_z", entry.DimZ)
	rec.Set("uploaded_by", entry.UploadedBy)

	// Store mods and materials as JSON
	if entry.Mods != nil {
		rec.Set("mods", entry.Mods)
	}
	if entry.ParsedMaterials != nil {
		rec.Set("materials", entry.ParsedMaterials)
	}

	// Attach the NBT file if data is available
	if len(entry.NBTData) > 0 {
		f, fErr := filesystem.NewFileFromBytes(entry.NBTData, entry.Filename)
		if fErr == nil {
			rec.Set("nbt_file", f)
		}
	}

	if err := app.Save(rec); err != nil {
		return "", ""
	}

	// Get the stored filename from the record (PB may rename it)
	storedName := rec.GetString("nbt_file")
	return rec.Id, storedName
}

// loadTempUploadPB tries to load a previously saved temp upload by token.
// Returns (entry, true) when found, otherwise (zero, false).
func loadTempUploadPB(app *pocketbase.PocketBase, token string) (tempUpload, bool) {
	var zero tempUpload
	if app == nil || token == "" {
		return zero, false
	}
	coll, err := app.FindCollectionByNameOrId("temp_uploads")
	if err != nil || coll == nil {
		return zero, false
	}
	recs, err := app.FindRecordsByFilter(coll.Id, "token = {:t}", "-created", 1, 0, dbx.Params{"t": token})
	if err != nil || len(recs) == 0 {
		return zero, false
	}
	r := recs[0]
	// best-effort mapping
	uploadedAt := r.GetDateTime("created").Time()

	// Load materials from JSON field
	var parsedMaterials []nbtparser.Material
	rawMaterials := r.Get("materials")
	if rawMaterials != nil {
		if b, err := json.Marshal(rawMaterials); err == nil {
			_ = json.Unmarshal(b, &parsedMaterials)
		}
	}

	// Load mods from JSON field
	var mods []string
	rawMods := r.Get("mods")
	if rawMods != nil {
		if b, err := json.Marshal(rawMods); err == nil {
			_ = json.Unmarshal(b, &mods)
		}
	}

	return tempUpload{
		Filename:        r.GetString("filename"),
		Size:            int64(r.GetInt("size")),
		Checksum:        r.GetString("checksum"),
		UploadedAt:      uploadedAt,
		ParsedSummary:   r.GetString("parsed_summary"),
		BlockCount:      r.GetInt("block_count"),
		DimX:            r.GetInt("dim_x"),
		DimY:            r.GetInt("dim_y"),
		DimZ:            r.GetInt("dim_z"),
		ParsedMaterials: parsedMaterials,
		Mods:            mods,
		UploadedBy:      r.GetString("uploaded_by"),
		PBRecordID:      r.Id,
		NBTStoredName:   r.GetString("nbt_file"),
	}, true
}
