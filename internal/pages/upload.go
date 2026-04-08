package pages

import (
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/mailer"
	"createmod/internal/models"
	"createmod/internal/moderation"
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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"createmod/internal/server"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/gosimple/slug"
	"github.com/sym01/htmlsanitizer"
)

// ModerationJobArgs contains the data needed to enqueue an async moderation job.
type ModerationJobArgs struct {
	SchematicID string
	Title       string
	Description string
	ImageURL    string
	Slug        string
}

// ModerationEnqueuer is a callback that enqueues a moderation job.
// Nil means no async moderation is available.
type ModerationEnqueuer func(ctx context.Context, args ModerationJobArgs) error

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
	AdditionalFiles   []tempUploadFile    // extra NBT files (variations/sets)
	PreUploadedImages []store.TempUploadImage // images uploaded via API
	TrustedUser       bool                // true if user has previously approved schematics
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
	IsUnclaimed      bool             // true if UploadedBy is empty (no owner yet)
	AdditionalFiles  []tempUploadFile // extra NBT files (variations/sets)
}

func UploadHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := UploadData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Upload"))
		d.UploadStep = 1
		d.Title = i18n.T(d.Language, "Upload A Schematic")
		d.Description = i18n.T(d.Language, "page.upload.description")
		d.Slug = "/upload"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.HideOutstream = true

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

// UploadPendingData holds data for the upload pending confirmation page.
type UploadPendingData struct {
	DefaultData
	SchematicName string
	SchematicURL  string
	SchematicID   string
	AutoApproved  bool
}

// UploadPendingHandler renders a simple moderation pending confirmation page.
// When called with HX-Request and an "id" param, returns a partial HTML fragment
// showing the current moderation status for HTMX polling.
func UploadPendingHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		schematicID := e.Request.URL.Query().Get("id")

		// HTMX poll: return moderation status fragment
		if schematicID != "" && e.Request.Header.Get("HX-Request") != "" {
			schem, err := appStore.Schematics.GetByID(e.Request.Context(), schematicID)
			if err != nil || schem == nil {
				return e.HTML(http.StatusOK, `<div id="moderation-status" class="text-secondary"><span class="spinner-border spinner-border-sm me-2"></span>Checking moderation status...</div>`)
			}

			name := e.Request.URL.Query().Get("name")
			schematicURL := ""
			if name != "" {
				schematicURL = "/schematics/" + name
			}

			if store.IsPublicState(schem.ModerationState) {
				return e.HTML(http.StatusOK, fmt.Sprintf(`<div id="moderation-status">
<div class="d-flex align-items-center mb-3">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-lg text-success me-2" width="32" height="32" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M5 12l5 5l10 -10" /></svg>
<span class="h3 mb-0">Your schematic has been published!</span>
</div>
<p>Your schematic passed moderation and is now live on the site.</p>
<div class="mt-3"><a href="%s" class="btn btn-primary">View Schematic</a></div>
</div>`, schematicURL))
			}
			if schem.ModerationState == store.ModerationFlagged || schem.ModerationState == store.ModerationRejected {
				return e.HTML(http.StatusOK, `<div id="moderation-status">
<div class="d-flex align-items-center mb-3">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-lg text-warning me-2" width="32" height="32" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 9v4" /><path d="M10.363 3.591l-8.106 13.534a1.914 1.914 0 0 0 1.636 2.871h16.214a1.914 1.914 0 0 0 1.636 -2.87l-8.106 -13.536a1.914 1.914 0 0 0 -3.274 0z" /><path d="M12 16h.01" /></svg>
<span class="h3 mb-0">Held for moderation</span>
</div>
<p>Your schematic is being held for additional moderation and will be published within 24 hours if accepted.</p>
</div>`)
			}

			// Still pending — keep polling
			return e.HTML(http.StatusOK, fmt.Sprintf(`<div id="moderation-status"
     hx-get="/upload/pending?id=%s&name=%s"
     hx-trigger="load delay:3s"
     hx-target="#moderation-status"
     hx-swap="outerHTML">
<span class="spinner-border spinner-border-sm me-2"></span>Checking moderation status...
</div>`, url.QueryEscape(schematicID), url.QueryEscape(name)))
		}

		d := UploadPendingData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Upload"), "/upload", i18n.T(d.Language, "Pending"))
		d.Title = i18n.T(d.Language, "Upload Pending Moderation")
		d.Description = i18n.T(d.Language, "page.upload_pending.description")
		d.Slug = "/upload/pending"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.HideOutstream = true

		// Read schematic name, ID, and auto-approved status from query params
		if name := e.Request.URL.Query().Get("name"); name != "" {
			d.SchematicName = name
			d.SchematicURL = "/schematics/" + name
		}
		d.SchematicID = schematicID
		d.AutoApproved = e.Request.URL.Query().Get("auto") == "true"

		html, err := registry.LoadFiles(uploadPendingTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadMakePublicHandler accepts POSTs to publish a previously uploaded temp schematic.
// Creates a real Schematic record, uploads images and copies NBT files from temp to
// schematics S3 prefix, handles additional files (variations), then cleans up temp data.
func UploadMakePublicHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service, moderationSvc *moderation.Service, mailService *mailer.Service, enqueueModeration ModerationEnqueuer) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}

		// Require authentication
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)

		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		ctx := e.Request.Context()

		// Verify the token exists and belongs to the authenticated user
		entry, err := appStore.TempUploads.GetByToken(ctx, token)
		if err != nil {
			return e.String(http.StatusNotFound, "invalid or expired token")
		}
		if entry.UploadedBy != "" && entry.UploadedBy != userID {
			return e.String(http.StatusForbidden, "you do not own this upload")
		}

		// Parse the multipart form (up to 100 MB in memory; the rest spills to
		// temp files).  Using ParseMultipartForm instead of ParseForm ensures
		// that the request body is fully consumed even when the client sends
		// file fields (featured_image, gallery).  Leaving the body unread
		// causes Go's HTTP server to drain or reset the connection, which
		// produces 502 errors through reverse proxies like Cloudflare.
		if err := e.Request.ParseMultipartForm(100 << 20); err != nil {
			slog.Error("make-public: failed to parse multipart form", "error", err)
			return e.String(http.StatusBadRequest, "failed to parse form")
		}

		// --- Parse form fields ---
		title := strings.TrimSpace(e.Request.FormValue("title"))
		if title == "" {
			title = strings.TrimSuffix(entry.Filename, ".nbt")
		}

		rawContent := e.Request.FormValue("content")
		if rawContent == "" {
			rawContent = e.Request.FormValue("description")
		}

		// Sanitize content
		sanitizer := htmlsanitizer.NewHTMLSanitizer()
		sanitizedContent, sErr := sanitizer.SanitizeString(rawContent)
		if sErr != nil {
			sanitizedContent = rawContent
		}

		// Generate excerpt
		excerpt := strip.StripTags(sanitizedContent)

		// Validate description quality
		if err := validateDescription(excerpt); err != nil {
			return e.BadRequestError(err.Error(), nil)
		}

		if len(excerpt) > 180 {
			excerpt = excerpt[:180]
		}

		// Resolve categories and tags
		var catIDs []string
		if rawCats := e.Request.Form["categories"]; len(rawCats) > 0 {
			catIDs = resolveCategoryIDs(ctx, appStore, rawCats)
		}
		var tagIDs []string
		if rawTags := e.Request.Form["tags"]; len(rawTags) > 0 {
			tagIDs = resolveTagIDs(ctx, appStore, rawTags)
		}

		// Version IDs
		var createmodVersionID *string
		if v := strings.TrimSpace(e.Request.FormValue("createmod_version")); v != "" {
			createmodVersionID = &v
		}
		var minecraftVersionID *string
		if v := strings.TrimSpace(e.Request.FormValue("minecraft_version")); v != "" {
			minecraftVersionID = &v
		}

		// Optional fields
		video := strings.TrimSpace(e.Request.FormValue("video"))
		paid := e.Request.FormValue("paid") == "true"
		externalURL := ""
		if paid {
			externalURL = strings.TrimSpace(e.Request.FormValue("external_url"))
		}

		// Scheduled publish
		var scheduledAt *time.Time
		if val := strings.TrimSpace(e.Request.FormValue("scheduled_at")); val != "" {
			when, perr := time.Parse(time.RFC3339, val)
			if perr != nil {
				const layout = "2006-01-02T15:04"
				if t2, err2 := time.ParseInLocation(layout, val, time.Local); err2 == nil {
					when = t2
					perr = nil
				}
			}
			if perr == nil && !when.IsZero() {
				utc := when.UTC()
				scheduledAt = &utc
			}
		}

		// Generate schematic ID and slug
		schematicID := generateSchematicID()
		nameSlug := slug.Make(title)
		if nameSlug == "" {
			nameSlug = schematicID
		}
		// Ensure the slug is unique among non-deleted schematics.
		if taken, _ := appStore.Schematics.NameExists(ctx, nameSlug); taken {
			nameSlug = makeUniqueSlug(ctx, appStore, nameSlug)
		}

		// --- Handle featured image ---
		var featuredFilename string

		// Check if featured image is a preloaded temp upload image
		if preloadedFeatured := strings.TrimSpace(e.Request.FormValue("featured_image_preloaded")); preloadedFeatured != "" {
			// Copy from temp uploads S3 to schematics S3
			srcKey := s3CollectionTempUploads + "/" + token + "/" + preloadedFeatured
			if storageSvc != nil {
				reader, dlErr := storageSvc.DownloadRaw(ctx, srcKey)
				if dlErr == nil {
					imgData, readErr := io.ReadAll(reader)
					_ = reader.Close()
					if readErr == nil {
						if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, preloadedFeatured, imgData, "image/webp"); uploadErr != nil {
							slog.Error("make-public: failed to copy preloaded featured image", "error", uploadErr)
						} else {
							featuredFilename = preloadedFeatured
						}
					}
				} else {
					slog.Error("make-public: failed to download preloaded featured image", "error", dlErr, "key", srcKey)
				}
			}
		}

		// Fall back to uploaded file if no preloaded featured image
		if featuredFilename == "" {
			if file, header, fileErr := e.Request.FormFile("featured_image"); fileErr == nil && header != nil {
				defer func() { _ = file.Close() }()

				if header.Size <= 5<<20 {
					ext := strings.ToLower(filepath.Ext(header.Filename))
					allowedImageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
					if allowedImageExts[ext] {
						data, readErr := io.ReadAll(file)
						if readErr == nil {
							data, filename, contentType := convertToWebP(data, sanitizeFilename(header.Filename))
							if storageSvc != nil {
								if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, filename, data, contentType); uploadErr != nil {
									slog.Error("make-public: failed to upload featured image", "error", uploadErr)
								} else {
									featuredFilename = filename
								}
							}
						}
					}
				}
			}
		}

		// --- Handle gallery images ---
		galleryFilenames := []string{}

		// Copy preloaded gallery images from temp uploads S3
		if preloadedGallery := e.Request.Form["preloaded_images"]; len(preloadedGallery) > 0 {
			for _, filename := range preloadedGallery {
				filename = strings.TrimSpace(filename)
				if filename == "" {
					continue
				}
				srcKey := s3CollectionTempUploads + "/" + token + "/" + filename
				if storageSvc != nil {
					reader, dlErr := storageSvc.DownloadRaw(ctx, srcKey)
					if dlErr != nil {
						slog.Error("make-public: failed to download preloaded gallery image", "error", dlErr, "key", srcKey)
						continue
					}
					imgData, readErr := io.ReadAll(reader)
					_ = reader.Close()
					if readErr != nil {
						continue
					}
					if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, filename, imgData, "image/webp"); uploadErr != nil {
						slog.Error("make-public: failed to copy preloaded gallery image", "error", uploadErr)
						continue
					}
					galleryFilenames = append(galleryFilenames, filename)
				}
			}
		}

		// Handle new file uploads for gallery
		if e.Request.MultipartForm != nil && e.Request.MultipartForm.File != nil {
			if galleryFiles, ok := e.Request.MultipartForm.File["gallery"]; ok && len(galleryFiles) > 0 {
				for _, fh := range galleryFiles {
					if fh.Size > 5<<20 {
						continue
					}
					ext := strings.ToLower(filepath.Ext(fh.Filename))
					allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
					if !allowedExts[ext] {
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
					data, filename, contentType := convertToWebP(data, sanitizeFilename(fh.Filename))
					if storageSvc != nil {
						if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, filename, data, contentType); uploadErr != nil {
							slog.Error("make-public: failed to upload gallery image", "error", uploadErr)
							continue
						}
					}
					galleryFilenames = append(galleryFilenames, filename)
				}
			}
		}

		// --- Copy NBT file from temp to schematics ---
		if entry.NbtS3Key != "" && storageSvc != nil {
			reader, dlErr := storageSvc.DownloadRaw(ctx, entry.NbtS3Key)
			if dlErr != nil {
				slog.Error("make-public: failed to download temp NBT", "error", dlErr, "key", entry.NbtS3Key)
				return e.String(http.StatusInternalServerError, "failed to retrieve uploaded file")
			}
			nbtData, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				slog.Error("make-public: failed to read temp NBT", "error", readErr)
				return e.String(http.StatusInternalServerError, "failed to read uploaded file")
			}
			if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, entry.Filename, nbtData, "application/octet-stream"); uploadErr != nil {
				slog.Error("make-public: failed to upload NBT to schematics", "error", uploadErr)
				return e.String(http.StatusInternalServerError, "failed to store schematic file")
			}
		}

		// --- Copy additional files (variations) from temp uploads ---
		tempFiles, _ := appStore.TempUploadFiles.ListByToken(ctx, token)
		for _, tf := range tempFiles {
			if tf.NbtS3Key == "" || storageSvc == nil {
				continue
			}
			reader, dlErr := storageSvc.DownloadRaw(ctx, tf.NbtS3Key)
			if dlErr != nil {
				slog.Error("make-public: failed to download additional file", "error", dlErr, "key", tf.NbtS3Key)
				continue
			}
			fileData, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil {
				slog.Error("make-public: failed to read additional file", "error", readErr)
				continue
			}

			// Upload to schematics S3 prefix
			s3Filename := schematicID + "_" + tf.Filename
			if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, s3Filename, fileData, "application/octet-stream"); uploadErr != nil {
				slog.Error("make-public: failed to upload additional file", "error", uploadErr)
				continue
			}

			// Create schematic_files record
			sf := &store.SchematicFile{
				SchematicID:  schematicID,
				Filename:     s3Filename,
				OriginalName: tf.Filename,
				Size:         tf.Size,
				MimeType:     "application/octet-stream",
			}
			if err := appStore.SchematicFiles.Create(ctx, sf); err != nil {
				slog.Error("make-public: failed to create schematic file record", "error", err)
			}
		}

		// --- Validate required featured image ---
		if featuredFilename == "" {
			return e.String(http.StatusBadRequest, "a schematic must have a featured image")
		}

		// Atomically mark as processing to prevent duplicate submissions.
		// This is placed after all validation so that a validation failure
		// does not lock the upload and block a corrected retry.
		if err := appStore.TempUploads.MarkProcessing(ctx, token); err != nil {
			return e.String(http.StatusConflict, "this upload is already being processed")
		}

		// --- Create schematic record ---
		now := time.Now()
		schem := &store.Schematic{
			ID:                 schematicID,
			AuthorID:           userID,
			Name:               nameSlug,
			Title:              title,
			Description:        sanitizedContent,
			Excerpt:            excerpt,
			Content:            sanitizedContent,
			Postdate:           &now,
			FeaturedImage:      featuredFilename,
			Gallery:            galleryFilenames,
			SchematicFile:      entry.Filename,
			Video:              video,
			CreatemodVersionID: createmodVersionID,
			MinecraftVersionID: minecraftVersionID,
			BlockCount:         entry.BlockCount,
			DimX:               entry.DimX,
			DimY:               entry.DimY,
			DimZ:               entry.DimZ,
			Materials:          entry.Materials,
			Mods:               entry.Mods,
			Paid:               paid,
			ExternalURL:        externalURL,
			ModerationState:    store.ModerationAutoReview,
			ScheduledAt:        scheduledAt,
		}

		if err := appStore.Schematics.Create(ctx, schem); err != nil {
			slog.Error("make-public: failed to create schematic", "error", err, "title", title)
			return e.String(http.StatusInternalServerError, "failed to create schematic")
		}

		// --- Set categories and tags ---
		if len(catIDs) > 0 {
			if err := appStore.Schematics.SetCategories(ctx, schem.ID, catIDs); err != nil {
				slog.Warn("make-public: failed to set categories", "error", err, "id", schem.ID)
			}
		}
		if len(tagIDs) > 0 {
			if err := appStore.Schematics.SetTags(ctx, schem.ID, tagIDs); err != nil {
				slog.Warn("make-public: failed to set tags", "error", err, "id", schem.ID)
			}
		}

		// --- Check trusted-user status ---
		// A user is trusted (auto-approved) only when they have at least 3
		// previously approved schematics AND zero soft-deleted schematics.
		autoApproved := false
		trustedUser := false
		authorCount, countErr := appStore.Schematics.CountByAuthor(ctx, userID)
		if countErr == nil && authorCount >= 3 {
			deletedCount, delErr := appStore.Schematics.CountSoftDeletedByAuthor(ctx, userID)
			if delErr == nil && deletedCount == 0 {
				trustedUser = true
			}
		}

		if trustedUser {
			// Trusted users skip moderation and are auto-approved
			schem.ModerationState = store.ModerationPublished
			if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("make-public: failed to auto-approve trusted user schematic", "error", updateErr, "id", schem.ID)
			} else {
				autoApproved = true
			}
		}

		// Always enqueue moderation job: for non-trusted users it runs moderation + language
		// detection; for trusted users moderation is skipped but language detection still runs.
		// The moderation job also checks the featured image for policy violations.
		if enqueueModeration != nil {
			if insertErr := enqueueModeration(ctx, ModerationJobArgs{
				SchematicID: schem.ID,
				Title:       title,
				Description: sanitizedContent,
				ImageURL:    featuredFilename,
				Slug:        nameSlug,
			}); insertErr != nil {
				slog.Error("make-public: failed to enqueue moderation job", "error", insertErr, "id", schem.ID)
			}
		}

		// Async image moderation for gallery images (featured is handled by the moderation job).
		moderateSchematicImages(moderationSvc, appStore, schem.ID, galleryFilenames)

		// --- Create NBT hash for duplicate detection ---
		if entry.Checksum != "" {
			if err := appStore.NBTHashes.Create(ctx, &store.NBTHash{
				ID:          generateSchematicID(),
				Hash:        entry.Checksum,
				SchematicID: &schem.ID,
				UploadedBy:  &userID,
			}); err != nil {
				slog.Error("make-public: failed to create NBT hash", "error", err, "id", schem.ID)
			}
		}

		// --- Send admin email notification ---
		// For non-trusted users with async moderation, the moderation worker
		// sends the email after moderation completes. Only send here for
		// trusted users (auto-approved) or when moderation is not available.
		sendAdminEmail := autoApproved || (moderationSvc == nil && mailService != nil)
		if sendAdminEmail && mailService != nil {
			emailTitle := schem.Title
			emailID := schem.ID
			emailName := nameSlug
			emailImage := featuredFilename
			go func() {
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "https://createmod.com"
				}
				var imageURL string
				if emailImage != "" {
					imageURL = fmt.Sprintf("%s/api/files/schematics/%s/%s", baseURL, emailID, url.PathEscape(emailImage))
				}
				schematicURL := fmt.Sprintf("%s/schematics/%s", baseURL, emailName)

				to := adminRecipients(appStore, mailService)
				if len(to) == 0 {
					return
				}
				from := mailService.DefaultFrom()

				var subject, bodyText string
				if autoApproved {
					subject = fmt.Sprintf("Schematic Auto-Approved: %s", emailTitle)
					bodyText = fmt.Sprintf("The schematic \"%s\" has been auto-approved and is now live on the site.", emailTitle)
				} else {
					subject = fmt.Sprintf("Schematic Needs Review: %s", emailTitle)
					bodyText = fmt.Sprintf("The schematic \"%s\" requires manual review before it can be published.", emailTitle)
				}

				body := mailer.SchematicEmailHTML(emailTitle, imageURL, schematicURL, bodyText)
				msg := &mailer.Message{From: from, To: to, Subject: subject, HTML: body}
				if err := mailService.Send(msg); err != nil {
					slog.Error("make-public: failed to send admin notification", "error", err)
				}
			}()
		}

		// --- Cleanup temp data ---
		// Delete temp upload files from S3
		for _, tf := range tempFiles {
			if tf.NbtS3Key != "" && storageSvc != nil {
				_ = storageSvc.DeleteRaw(ctx, tf.NbtS3Key)
			}
		}
		if entry.NbtS3Key != "" && storageSvc != nil {
			_ = storageSvc.DeleteRaw(ctx, entry.NbtS3Key)
		}
		// Delete temp upload images from S3 and DB
		if tempImages, imgErr := appStore.TempUploadImages.ListByToken(ctx, token); imgErr == nil {
			for _, img := range tempImages {
				if img.S3Key != "" && storageSvc != nil {
					_ = storageSvc.DeleteRaw(ctx, img.S3Key)
				}
			}
		}
		_ = appStore.TempUploadImages.DeleteByToken(ctx, token)
		// Delete temp upload file records
		_ = appStore.TempUploadFiles.DeleteByToken(ctx, token)
		// Delete temp upload record
		_ = appStore.TempUploads.Delete(ctx, token)

		pendingURL := fmt.Sprintf("/upload/pending?name=%s&id=%s&auto=%t", url.QueryEscape(nameSlug), url.QueryEscape(schem.ID), autoApproved)
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, pendingURL))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, pendingURL))
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
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Upload"), "/upload", i18n.T(d.Language, "Preview"))
		d.Title = i18n.T(d.Language, "Schematic Review")
		d.Description = i18n.T(d.Language, "page.upload_review.description")
		d.Slug = "/u/" + token
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.HideOutstream = true
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
		d.IsUnclaimed = entry.UploadedBy == ""
		d.AdditionalFiles = additionalFiles
		h, err := registry.LoadFiles(uploadPreviewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, h)
	}
}

// UploadClaimHandler allows an authenticated user to claim an unclaimed temp upload.
// Uses an atomic conditional UPDATE (uploaded_by = '' guard) to prevent race conditions
// and ensure a claimed upload cannot be stolen by another user.
// POST /u/{token}/claim
func UploadClaimHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.String(http.StatusBadRequest, "missing token")
		}

		userID := authenticatedUserID(e)
		if err := appStore.TempUploads.Claim(e.Request.Context(), token, userID); err != nil {
			return e.String(http.StatusConflict, "already claimed")
		}

		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/u/"+token))
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

// uploadImageResponse describes a single image in the upload API response.
type uploadImageResponse struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
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
	Images     []uploadImageResponse `json:"images,omitempty"`
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

		// Sanitize the filename to be ASCII-safe for URLs and S3 keys
		safeFilename := sanitizeFilename(header.Filename)

		// Upload NBT file to S3
		nbtS3Key := s3CollectionTempUploads + "/" + token + "/" + safeFilename
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
			slog.Error("failed to persist temp upload", "error", err, "token", token)
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save upload metadata"})
		}

		// Build file URL — the filename is already sanitized to ASCII-safe characters
		fileURL := "/api/files/" + s3CollectionTempUploads + "/" + token + "/" + url.PathEscape(safeFilename)

		// Build JSON response
		resp := uploadNBTResponse{
			Token:      token,
			URL:        "/u/" + token,
			Checksum:   checksum,
			Filename:   safeFilename,
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

// generateSchematicID generates a random 15-character hex ID matching the existing ID format.
func generateSchematicID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:15]
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

// makeUniqueSlug appends random characters to baseSlug until a unique name is
// found among non-deleted schematics. It tries up to 30 times, starting with a
// single random character and growing the suffix length on each attempt.
func makeUniqueSlug(ctx context.Context, appStore *store.Store, baseSlug string) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < 30; i++ {
		// Generate a random suffix: length grows as attempts increase.
		suffixLen := i/len(chars) + 1
		suffix := make([]byte, suffixLen)
		for j := range suffix {
			b := make([]byte, 1)
			_, _ = rand.Read(b)
			suffix[j] = chars[int(b[0])%len(chars)]
		}
		candidate := baseSlug + "-" + string(suffix)
		if taken, _ := appStore.Schematics.NameExists(ctx, candidate); !taken {
			return candidate
		}
	}
	// Final fallback: append a full random hex string.
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return baseSlug + "-" + hex.EncodeToString(b)
}
