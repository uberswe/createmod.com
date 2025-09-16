package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/nbtparser"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const uploadPendingTemplate = "./template/upload_pending.html"

type tempUpload struct {
	Filename      string
	Size          int64
	Checksum      string
	UploadedAt    time.Time
	ParsedSummary string
	BlockCount    int
	Materials     []string
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

var uploadTemplates = append([]string{
	uploadTemplate,
}, commonTemplates...)

var uploadPendingTemplates = append([]string{
	uploadPendingTemplate,
}, commonTemplates...)

const uploadPreviewTemplate = "./template/upload_preview.html"

var uploadPreviewTemplates = append([]string{
	uploadPreviewTemplate,
}, commonTemplates...)

type UploadData struct {
	DefaultData
	MinecraftVersions []models.MinecraftVersion
	CreatemodVersions []models.CreatemodVersion
	Tags              []models.SchematicTag
}

type UploadPreviewData struct {
	DefaultData
	Token         string
	Filename      string
	Size          int64
	Checksum      string
	UploadedAt    time.Time
	ParsedSummary string
	BlockCount    int
	Materials     []string
}

func UploadHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := UploadData{}
		d.Populate(e)
		d.Title = "Upload A Schematic"
		d.Description = "Upload a Create Mod schematic to share it with others."
		d.Slug = "/upload"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
		d.Tags = allTags(app)
		d.MinecraftVersions = allMinecraftVersions(app)
		d.CreatemodVersions = allCreatemodVersions(app)
		html, err := registry.LoadFiles(uploadTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadPendingHandler renders a simple moderation pending confirmation page.
func UploadPendingHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		d := DefaultData{}
		d.Populate(e)
		d.Title = "Upload Pending Moderation"
		d.Description = "Your schematic has been queued for moderation and will appear soon."
		d.Slug = "/upload/pending"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategories(app, cacheService)
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
func UploadMakePublicHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
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
			e.Response.Header().Set("HX-Redirect", "/upload/pending")
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, "/upload/pending")
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
func UploadPreviewHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
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
		// Render template with default data
		d := UploadPreviewData{}
		d.Populate(e)
		d.Title = "Private Preview"
		d.Description = "Private preview of your uploaded schematic. Share this link with anyone to allow viewing."
		d.Slug = "/u/" + token
		d.Categories = allCategories(app, cacheService)
		d.Token = token
		d.Filename = entry.Filename
		d.Size = entry.Size
		d.Checksum = entry.Checksum
		d.UploadedAt = entry.UploadedAt
		d.ParsedSummary = entry.ParsedSummary
		d.BlockCount = entry.BlockCount
		d.Materials = entry.Materials
		h, err := registry.LoadFiles(uploadPreviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, h)
	}
}

// UploadNBTHandler is a placeholder for Upload step 1 (server-side NBT upload & validation).
// It will be implemented to parse the uploaded NBT, validate via mcnbt, hash for dup detection,
// and create a temporary/private upload record before returning a preview URL.
func UploadNBTHandler(app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Best-effort parse of multipart form; actual limits/validation will be added later.
		_ = e.Request.ParseMultipartForm(128 << 20) // 128 MB limit for now
		// Attempt to read the file field (common names: "nbt" or "file").
		file, header, err := e.Request.FormFile("nbt")
		if err != nil {
			file, header, err = e.Request.FormFile("file")
			if err != nil {
				return e.String(http.StatusBadRequest, "missing NBT file in form (expected field 'nbt' or 'file')")
			}
		}
		if file != nil {
			defer file.Close()
		}
		// Basic filename validation before parsing
		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return e.String(http.StatusBadRequest, "invalid file type: expected .nbt")
		}
		// Read the uploaded file fully into memory to compute size and checksum
		data, err := io.ReadAll(file)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to read uploaded file")
		}
		// Minimal backend validation (scaffold for mcnbt integration)
		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			return e.String(http.StatusBadRequest, msg)
		}
		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection via PocketBase (temp_uploads and schematics) — best effort.
		// If the collections exist and we find a record with the same checksum,
		// consider it a duplicate and return 409 with a helpful message.
		if app != nil {
			// Check temp_uploads first
			if coll, err := app.FindCollectionByNameOrId("temp_uploads"); err == nil && coll != nil {
				recs, err := app.FindRecordsByFilter(coll.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
				if err == nil && len(recs) > 0 {
					return e.String(http.StatusConflict, "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation, otherwise it may be blacklisted by the original creator. If you need more help contact us: /contact")
				}
			}
			// Then check existing schematics (if the collection and field exist)
			if schemColl, err := app.FindCollectionByNameOrId("schematics"); err == nil && schemColl != nil {
				recs, err := app.FindRecordsByFilter(schemColl.Id, "checksum = {:c}", "-created", 1, 0, dbx.Params{"c": checksum})
				if err == nil && len(recs) > 0 {
					return e.String(http.StatusConflict, "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation, otherwise it may be blacklisted by the original creator. If you need more help contact us: /contact")
				}
			}
		}

		// Duplicate detection (in-memory temp store): if an existing temp upload has the same checksum,
		// reject this upload with a helpful message.
		tempUploadStore.RLock()
		for _, entry := range tempUploadStore.m {
			if entry.Checksum == checksum {
				tempUploadStore.RUnlock()
				return e.String(http.StatusConflict, "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation, otherwise it may be blacklisted by the original creator. If you need more help contact us: /contact")
			}
		}
		tempUploadStore.RUnlock()

		// Best-effort: persist hash to nbt_hashes (for bulk/duplicate tracking)
		if app != nil {
			if coll, err := app.FindCollectionByNameOrId("nbt_hashes"); err == nil && coll != nil {
				rec := core.NewRecord(coll)
				rec.Set("checksum", checksum)
				if e.Auth != nil {
					rec.Set("uploaded_by", e.Auth.Id)
				}
				_ = app.Save(rec) // ignore errors (e.g., duplicate)
			}
		}

		// Generate a simple random token and return a preview URL.
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return e.String(http.StatusInternalServerError, "failed to generate preview token")
		}
		token := hex.EncodeToString(buf)
		// Try to produce a parsed summary via nbtparser (future mcnbt integration).
		// Gracefully fall back to a forward-compatible placeholder if parsing is unavailable.
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = fmt.Sprintf("size=%d bytes; nbt=unparsed", n)
		}
		// Extract basic stats (stubbed for now)
		blockCount, materials, _ := nbtparser.ExtractStats(data)
		// Store metadata in the in-memory temporary store
		tempUploadStore.Lock()
		tempUploadStore.m[token] = tempUpload{
			Filename:      header.Filename,
			Size:          n,
			Checksum:      checksum,
			UploadedAt:    time.Now(),
			ParsedSummary: parsed,
			BlockCount:    blockCount,
			Materials:     materials,
		}
		entry := tempUploadStore.m[token]
		tempUploadStore.Unlock()
		// Best-effort: persist to PocketBase so the token link survives restarts
		persistTempUploadPB(app, token, entry)
		previewURL := "/u/" + token
		return e.String(http.StatusOK, fmt.Sprintf("ok: sha256=%s token=%s url=%s", checksum, token, previewURL))
	}
}

// --- Persistent storage helpers (best-effort PocketBase) ---
// persistTempUploadPB attempts to save the temporary upload entry in the
// "temp_uploads" collection. If the collection doesn't exist, it silently
// returns without error so local/dev and tests without migrations keep working.
func persistTempUploadPB(app *pocketbase.PocketBase, token string, entry tempUpload) {
	if app == nil {
		return
	}
	coll, err := app.FindCollectionByNameOrId("temp_uploads")
	if err != nil || coll == nil {
		return
	}
	rec := core.NewRecord(coll)
	rec.Set("token", token)
	rec.Set("filename", entry.Filename)
	rec.Set("size", entry.Size)
	rec.Set("checksum", entry.Checksum)
	rec.Set("parsed_summary", entry.ParsedSummary)
	_ = app.Save(rec)
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
	return tempUpload{
		Filename:      r.GetString("filename"),
		Size:          int64(r.GetInt("size")),
		Checksum:      r.GetString("checksum"),
		UploadedAt:    uploadedAt,
		ParsedSummary: r.GetString("parsed_summary"),
	}, true
}
