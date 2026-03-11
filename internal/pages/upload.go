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
	TrustedUser       bool             // true if user has previously approved schematics
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

// UploadPendingData holds data for the upload pending confirmation page.
type UploadPendingData struct {
	DefaultData
	SchematicName string
	SchematicURL  string
	AutoApproved  bool
}

// UploadPendingHandler renders a simple moderation pending confirmation page.
func UploadPendingHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := UploadPendingData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Upload Pending Moderation")
		d.Description = i18n.T(d.Language, "page.upload_pending.description")
		d.Slug = "/upload/pending"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// Read schematic name and auto-approved status from query params
		if name := e.Request.URL.Query().Get("name"); name != "" {
			d.SchematicName = name
			d.SchematicURL = "/schematics/" + name
		}
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
func UploadMakePublicHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service, moderationSvc *moderation.Service, mailService *mailer.Service) func(e *server.RequestEvent) error {
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

		// --- Handle featured image ---
		var featuredFilename string
		if file, header, fileErr := e.Request.FormFile("featured_image"); fileErr == nil && header != nil {
			defer func() { _ = file.Close() }()

			if header.Size <= 5<<20 {
				ext := strings.ToLower(filepath.Ext(header.Filename))
				allowedImageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
				if allowedImageExts[ext] {
					data, readErr := io.ReadAll(file)
					if readErr == nil {
						data, filename, contentType := convertToWebP(data, header.Filename)
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

		// --- Handle gallery images ---
		galleryFilenames := []string{}
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
					data, filename, contentType := convertToWebP(data, fh.Filename)
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
			Moderated:          false,
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

		// --- Check trusted-user status (has previously approved schematics) ---
		autoApproved := false
		trustedUser := false
		authorCount, countErr := appStore.Schematics.CountByAuthor(ctx, userID)
		if countErr == nil && authorCount > 0 {
			trustedUser = true
		}

		if trustedUser {
			// Trusted users skip moderation and are auto-approved
			schem.Moderated = true
			if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
				slog.Error("make-public: failed to auto-approve trusted user schematic", "error", updateErr, "id", schem.ID)
			} else {
				autoApproved = true
			}
		} else if moderationSvc != nil {
			policyResult, policyErr := moderationSvc.CheckSchematic(title, sanitizedContent, "")
			if policyErr != nil {
				slog.Warn("make-public: moderation policy check unavailable, manual review required", "error", policyErr, "id", schem.ID)
			} else if !policyResult.Approved {
				schem.Blacklisted = true
				schem.ModerationReason = policyResult.Reason
				if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
					slog.Error("make-public: failed to blacklist schematic", "error", updateErr, "id", schem.ID)
				}
			} else {
				// Policy passed, check quality
				qualityResult, qualityErr := moderationSvc.CheckSchematicQuality(title, sanitizedContent)
				if qualityErr != nil {
					slog.Warn("make-public: moderation quality check unavailable, manual review required", "error", qualityErr, "id", schem.ID)
				} else if !qualityResult.Approved {
					schem.Blacklisted = true
					schem.ModerationReason = qualityResult.Reason
					if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
						slog.Error("make-public: failed to blacklist schematic", "error", updateErr, "id", schem.ID)
					}
				} else {
					// Both checks passed — auto-approve
					schem.Moderated = true
					if updateErr := appStore.Schematics.Update(ctx, schem); updateErr != nil {
						slog.Error("make-public: failed to auto-approve schematic", "error", updateErr, "id", schem.ID)
					} else {
						autoApproved = true
					}
				}
			}
		}

		// --- Create NBT hash for duplicate detection ---
		if entry.Checksum != "" {
			_ = appStore.NBTHashes.Create(ctx, &store.NBTHash{
				Hash:        entry.Checksum,
				SchematicID: &schem.ID,
				UploadedBy:  &userID,
			})
		}

		// --- Send admin email notification ---
		if mailService != nil {
			emailTitle := schem.Title
			emailID := schem.ID
			emailName := nameSlug
			emailImage := featuredFilename
			emailBlacklisted := schem.Blacklisted
			emailReason := schem.ModerationReason
			go func() {
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "https://createmod.com"
				}
				var imageURL string
				if emailImage != "" {
					imageURL = fmt.Sprintf("%s/api/files/schematics/%s/%s", baseURL, emailID, emailImage)
				}
				schematicURL := fmt.Sprintf("%s/schematics/%s", baseURL, emailName)

				to := adminRecipients(appStore, mailService)
				if len(to) == 0 {
					return
				}
				from := mailService.DefaultFrom()

				var subject, bodyText string
				if emailBlacklisted {
					subject = fmt.Sprintf("Schematic Blocked: %s", emailTitle)
					bodyText = fmt.Sprintf("The schematic \"%s\" was blocked by automated moderation. Reason: %s", emailTitle, emailReason)
				} else if autoApproved {
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
		// Delete temp upload file records
		_ = appStore.TempUploadFiles.DeleteByToken(ctx, token)
		// Delete temp upload record
		_ = appStore.TempUploads.Delete(ctx, token)

		pendingURL := fmt.Sprintf("/upload/pending?name=%s&auto=%t", url.QueryEscape(nameSlug), autoApproved)
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
