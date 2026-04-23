package pages

import (
	"bufio"
	"bytes"
	"context"
	"createmod/internal/cache"
	"createmod/internal/moderation"
	"createmod/internal/nbtparser"
	"createmod/internal/storage"
	"createmod/internal/store"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"createmod/internal/server"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/sunshineplan/imgconv"
	"github.com/sym01/htmlsanitizer"
)

// s3CollectionSchematics is the S3 prefix for schematic files.
var s3CollectionSchematics = storage.CollectionPrefix("schematics")

// SchematicUpdateHandler handles POST /schematics/{id}/update to update a schematic.
// Requires authentication. The authenticated user must be the schematic author.
// Accepts multipart form data with optional file uploads for schematic_file, featured_image, and gallery.
// Creates a version snapshot of the previous state before applying updates.
func SchematicUpdateHandler(
	cacheService *cache.Service,
	storageSvc *storage.Service,
	appStore *store.Store,
	moderationSvc *moderation.Service,
) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("authentication required", nil)
		}

		schematicID := e.Request.PathValue("id")
		if schematicID == "" {
			return e.BadRequestError("schematic id is required", nil)
		}

		ctx := context.Background()

		// Fetch the existing schematic
		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return e.NotFoundError("schematic not found", nil)
		}

		// Author check
		if schem.AuthorID != userID {
			return e.ForbiddenError("you are not the author of this schematic", nil)
		}

		// Parse multipart form (up to 32 MB for images + schematic file)
		if err := e.Request.ParseMultipartForm(32 << 20); err != nil {
			return e.BadRequestError("invalid form data", nil)
		}

		// --- Capture previous state for version snapshot ---
		prevSnapshot := buildSchematicSnapshot(schem)

		// --- Read form fields ---
		title := strings.TrimSpace(e.Request.FormValue("title"))
		content := e.Request.FormValue("content")
		video := strings.TrimSpace(e.Request.FormValue("video"))
		categories := resolveCategoryIDs(ctx, appStore, e.Request.Form["categories"])
		tags := resolveTagIDs(ctx, appStore, e.Request.Form["tags"])
		createmodVersion := strings.TrimSpace(e.Request.FormValue("createmod_version"))
		minecraftVersion := strings.TrimSpace(e.Request.FormValue("minecraft_version"))
		removeImagesRaw := strings.TrimSpace(e.Request.FormValue("remove_images"))
		removeFeaturedImage := e.Request.FormValue("remove_featured_image") == "true"
		setFeaturedFromGallery := strings.TrimSpace(e.Request.FormValue("set_featured_from_gallery"))

		// --- Apply text field updates ---
		if title != "" {
			schem.Title = title
		}

		if content != "" {
			// Sanitize HTML content
			sanitizer := htmlsanitizer.NewHTMLSanitizer()
			sanitizedContent, sanitizeErr := sanitizer.SanitizeString(content)
			if sanitizeErr != nil {
				// Fallback: use the raw content (logged for investigation)
				slog.Warn("schematic update: HTML sanitization failed, using raw content", "error", sanitizeErr, "id", schematicID)
				sanitizedContent = content
			}
			schem.Content = sanitizedContent

			// Regenerate excerpt from content
			plainText := strip.StripTags(sanitizedContent)

			// Validate description quality
			if err := validateDescription(plainText); err != nil {
				return e.BadRequestError(err.Error(), nil)
			}

			if len(plainText) > 180 {
				schem.Excerpt = plainText[:180] + "..."
			} else {
				schem.Excerpt = plainText
			}
		}

		if video != "" || e.Request.FormValue("video") != "" {
			schem.Video = video
		}

		// --- Paid / External URL ---
		if paidStr := e.Request.FormValue("paid"); paidStr != "" {
			schem.Paid = paidStr == "true"
		}
		if externalURL := strings.TrimSpace(e.Request.FormValue("external_url")); externalURL != "" {
			schem.ExternalURL = externalURL
		}
		if !schem.Paid {
			schem.ExternalURL = ""
		}
		if schem.Paid && schem.ExternalURL == "" {
			return e.BadRequestError("paid schematics require an external URL", nil)
		}

		if createmodVersion != "" {
			schem.CreatemodVersionID = &createmodVersion
		} else if e.Request.FormValue("createmod_version") != "" {
			// Explicitly cleared
			schem.CreatemodVersionID = nil
		}

		if minecraftVersion != "" {
			schem.MinecraftVersionID = &minecraftVersion
		} else if e.Request.FormValue("minecraft_version") != "" {
			// Explicitly cleared
			schem.MinecraftVersionID = nil
		}

		// --- Handle schematic file upload (optional) ---
		if file, header, fileErr := e.Request.FormFile("schematic_file"); fileErr == nil && header != nil {
			defer func() { _ = file.Close() }()

			if !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
				return e.BadRequestError("schematic file must be .nbt format", nil)
			}
			if header.Size > maxUploadSize {
				return e.BadRequestError("schematic file too large (max 10MB)", nil)
			}

			data, readErr := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
			if readErr != nil {
				return e.InternalServerError("failed to read schematic file", nil)
			}
			if int64(len(data)) > maxUploadSize {
				return e.BadRequestError("schematic file too large (max 10MB)", nil)
			}

			// Validate NBT
			if ok, reason := nbtparser.Validate(data); !ok {
				msg := "invalid NBT file"
				if reason != "" {
					msg += ": " + reason
				}
				return e.BadRequestError(msg, nil)
			}

			// Extract stats from NBT
			blockCount, _, _ := nbtparser.ExtractStats(data)
			dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)
			parsedMaterials, _ := nbtparser.ExtractMaterials(data)

			schem.BlockCount = blockCount
			schem.DimX = dimX
			schem.DimY = dimY
			schem.DimZ = dimZ

			if parsedMaterials != nil {
				materialsJSON, _ := json.Marshal(parsedMaterials)
				schem.Materials = materialsJSON
			}

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
			modsJSON, _ := json.Marshal(mods)
			schem.Mods = modsJSON

			// Upload to S3
			filename := sanitizeFilename(header.Filename)
			if storageSvc != nil {
				if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, filename, data, "application/octet-stream"); uploadErr != nil {
					slog.Error("schematic update: failed to upload NBT to S3", "error", uploadErr, "id", schematicID)
					return e.InternalServerError("failed to store schematic file", nil)
				}
			}
			schem.SchematicFile = filename
		}

		// Track newly uploaded images for async moderation.
		var newImageFilenames []string

		// --- Handle featured image removal ---
		if removeFeaturedImage && schem.FeaturedImage != "" {
			if storageSvc != nil {
				if delErr := storageSvc.Delete(ctx, s3CollectionSchematics, schematicID, schem.FeaturedImage); delErr != nil {
					slog.Warn("schematic update: failed to delete old featured image from S3", "error", delErr, "id", schematicID)
				}
			}
			schem.FeaturedImage = ""
		}

		// --- Handle featured image upload (optional) ---
		if file, header, fileErr := e.Request.FormFile("featured_image"); fileErr == nil && header != nil {
			defer func() { _ = file.Close() }()

			if header.Size > 5<<20 { // 5 MB limit for images
				return e.BadRequestError("featured image too large (max 5MB)", nil)
			}

			ext := strings.ToLower(filepath.Ext(header.Filename))
			allowedImageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
			if !allowedImageExts[ext] {
				return e.BadRequestError("featured image must be jpg, png, webp, or gif", nil)
			}

			data, readErr := io.ReadAll(file)
			if readErr != nil {
				return e.InternalServerError("failed to read featured image", nil)
			}

			oldFeatured := schem.FeaturedImage
			data, filename, contentType := convertToWebP(data, sanitizeFilename(header.Filename))
			if storageSvc != nil {
				if uploadErr := storageSvc.UploadBytes(ctx, s3CollectionSchematics, schematicID, filename, data, contentType); uploadErr != nil {
					slog.Error("schematic update: failed to upload featured image to S3", "error", uploadErr, "id", schematicID)
					return e.InternalServerError("failed to store featured image", nil)
				}
			}
			// Demote old featured image to gallery instead of deleting it
			if oldFeatured != "" && oldFeatured != filename {
				schem.Gallery = append([]string{oldFeatured}, schem.Gallery...)
			}
			schem.FeaturedImage = filename
			newImageFilenames = append(newImageFilenames, filename)
		}

		// --- Promote gallery image to featured ---
		if setFeaturedFromGallery != "" {
			// Verify the filename exists in the current gallery
			found := false
			for _, fn := range schem.Gallery {
				if fn == setFeaturedFromGallery {
					found = true
					break
				}
			}
			if found {
				oldFeatured := schem.FeaturedImage
				schem.FeaturedImage = setFeaturedFromGallery
				// Remove promoted image from gallery, add old featured to gallery if it existed
				newGallery := make([]string, 0, len(schem.Gallery))
				for _, fn := range schem.Gallery {
					if fn == setFeaturedFromGallery {
						continue
					}
					newGallery = append(newGallery, fn)
				}
				if oldFeatured != "" {
					newGallery = append([]string{oldFeatured}, newGallery...)
				}
				schem.Gallery = newGallery
			}
		}

		// --- Handle gallery image removal ---
		var removeSet map[string]bool
		if removeImagesRaw != "" {
			parts := strings.Split(removeImagesRaw, ",")
			removeSet = make(map[string]bool, len(parts))
			for _, p := range parts {
				if fn := strings.TrimSpace(p); fn != "" {
					removeSet[fn] = true
				}
			}
		}

		// Start with existing gallery, removing any marked for deletion
		galleryFilenames := make([]string, 0, len(schem.Gallery))
		for _, fn := range schem.Gallery {
			if removeSet[fn] {
				// Delete removed gallery image from S3
				if storageSvc != nil {
					if delErr := storageSvc.Delete(ctx, s3CollectionSchematics, schematicID, fn); delErr != nil {
						slog.Warn("schematic update: failed to delete gallery image from S3", "error", delErr, "id", schematicID, "file", fn)
					}
				}
				continue
			}
			galleryFilenames = append(galleryFilenames, fn)
		}

		// --- Handle gallery uploads (optional, multiple files) ---
		if e.Request.MultipartForm != nil && e.Request.MultipartForm.File != nil {
			if galleryFiles, ok := e.Request.MultipartForm.File["gallery"]; ok && len(galleryFiles) > 0 {
				for _, fh := range galleryFiles {
					if fh.Size > 5<<20 { // 5 MB per image
						continue // skip oversized files
					}
					ext := strings.ToLower(filepath.Ext(fh.Filename))
					allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
					if !allowedExts[ext] {
						continue // skip unsupported formats
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
							slog.Error("schematic update: failed to upload gallery image to S3", "error", uploadErr, "id", schematicID)
							continue
						}
					}
					galleryFilenames = append(galleryFilenames, filename)
					newImageFilenames = append(newImageFilenames, filename)
				}
			}
		}

		// Update gallery if anything changed (removals or additions)
		schem.Gallery = galleryFilenames

		// --- Update modified timestamp ---
		now := time.Now()
		schem.Modified = &now
		schem.Updated = now

		// --- Persist the schematic update ---
		if err := appStore.Schematics.Update(ctx, schem); err != nil {
			slog.Error("schematic update: failed to update", "error", err, "id", schematicID)
			return e.InternalServerError("failed to update schematic", nil)
		}

		// --- Update categories and tags ---
		if len(categories) > 0 {
			if err := appStore.Schematics.SetCategories(ctx, schematicID, categories); err != nil {
				slog.Warn("schematic update: failed to set categories", "error", err, "id", schematicID)
			}
		}
		if err := appStore.Schematics.SetTags(ctx, schematicID, tags); err != nil {
			slog.Warn("schematic update: failed to set tags", "error", err, "id", schematicID)
		}

		// --- Async image moderation for any newly uploaded images ---
		moderateSchematicImages(moderationSvc, appStore, schematicID, newImageFilenames)

		// --- Create version snapshot ---
		createVersionSnapshot(appStore, schematicID, prevSnapshot, schem)

		// --- Clear cache ---
		cacheService.DeleteSchematic(cache.SchematicKey(schematicID))
		cacheService.DeleteSchematicHTML(schem.Name)
		cacheService.DeleteSchematicsListHTML()
		RefreshIndexCache(cacheService, appStore, []int{7})

		// --- Respond ---
		if e.Request.Header.Get("HX-Request") != "" {
			dest := "/schematics/" + schem.Name
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status": "ok",
			"id":     schematicID,
			"name":   schem.Name,
		})
	}
}

// SchematicDeleteHandler handles DELETE /schematics/{id} to soft-delete a schematic.
// Requires authentication. The authenticated user must be the schematic author.
func SchematicDeleteHandler(
	cacheService *cache.Service,
	appStore *store.Store,
) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		userID := authenticatedUserID(e)
		if userID == "" {
			return e.UnauthorizedError("authentication required", nil)
		}

		schematicID := e.Request.PathValue("id")
		if schematicID == "" {
			return e.BadRequestError("schematic id is required", nil)
		}

		ctx := context.Background()

		// Fetch the existing schematic
		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return e.NotFoundError("schematic not found", nil)
		}

		// Author check
		if schem.AuthorID != userID {
			return e.ForbiddenError("you are not the author of this schematic", nil)
		}

		// Soft-delete
		if err := appStore.Schematics.SoftDelete(ctx, schematicID); err != nil {
			slog.Error("schematic delete: failed to soft-delete", "error", err, "id", schematicID)
			return e.InternalServerError("failed to delete schematic", nil)
		}

		// Clear cache
		cacheService.DeleteSchematic(cache.SchematicKey(schematicID))
		cacheService.DeleteSchematicHTML(schem.Name)
		cacheService.DeleteSchematicsListHTML()
		RefreshIndexCache(cacheService, appStore, []int{7})

		// Respond
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, "/schematics"))
			return e.HTML(http.StatusNoContent, "")
		}

		return e.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"id":     schematicID,
		})
	}
}

// convertToWebP converts image data to WebP format. GIF files are skipped (may be animated).
// Returns the (possibly converted) data, filename, and content type.
// Falls back to the original if conversion fails.
func convertToWebP(data []byte, filename string) ([]byte, string, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".gif" || ext == ".webp" {
		return data, filename, http.DetectContentType(data)
	}

	img, err := imgconv.Decode(bytes.NewReader(data))
	if err != nil {
		return data, filename, http.DetectContentType(data)
	}

	var out bytes.Buffer
	bw := bufio.NewWriter(&out)
	if err := imgconv.Write(bw, img, &imgconv.FormatOption{
		Format:       imgconv.WEBP,
		EncodeOption: []imgconv.EncodeOption{imgconv.Quality(80)},
	}); err != nil {
		return data, filename, http.DetectContentType(data)
	}
	_ = bw.Flush()

	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	return out.Bytes(), baseName + ".webp", "image/webp"
}

// buildSchematicSnapshot captures the current state of a schematic for version history.
func buildSchematicSnapshot(s *store.Schematic) map[string]any {
	cmVersionID := ""
	if s.CreatemodVersionID != nil {
		cmVersionID = *s.CreatemodVersionID
	}
	mcVersionID := ""
	if s.MinecraftVersionID != nil {
		mcVersionID = *s.MinecraftVersionID
	}
	postdate := s.Created
	if s.Postdate != nil {
		postdate = *s.Postdate
	}

	return map[string]any{
		"title":             s.Title,
		"content":           s.Content,
		"excerpt":           s.Excerpt,
		"featured_image":    s.FeaturedImage,
		"gallery":           s.Gallery,
		"video":             s.Video,
		"has_dependencies":  s.HasDependencies,
		"dependencies":      s.Dependencies,
		"createmod_version": cmVersionID,
		"minecraft_version": mcVersionID,
		"paid":              s.Paid,
		"schematic_file":    s.SchematicFile,
		"postdate":          postdate,
		"updated":           s.Updated,
	}
}

// createVersionSnapshot persists a version snapshot and computes a diff note.
func createVersionSnapshot(appStore *store.Store, schematicID string, prevSnapshot map[string]any, newSchem *store.Schematic) {
	ctx := context.Background()

	data, err := json.Marshal(prevSnapshot)
	if err != nil {
		slog.Warn("schematic update: failed to marshal version snapshot", "error", err, "id", schematicID)
		return
	}

	// Compute changed fields
	changed := computeSchematicDiff(prevSnapshot, newSchem)
	note := ""
	if len(changed) > 0 {
		note = "Fields changed: " + strings.Join(changed, ", ")
	}

	verNum := 1
	if latest, err := appStore.Versions.GetLatestVersion(ctx, schematicID); err == nil {
		verNum = latest + 1
	}

	if err := appStore.Versions.Create(ctx, &store.SchematicVersion{
		SchematicID: schematicID,
		Version:     verNum,
		Snapshot:    string(data),
		Note:        note,
	}); err != nil {
		slog.Warn("schematic update: failed to save version snapshot", "error", err, "id", schematicID)
	}
}

// computeSchematicDiff compares a previous snapshot (map) with the new schematic state
// and returns a list of changed field names.
func computeSchematicDiff(prev map[string]any, newSchem *store.Schematic) []string {
	changed := make([]string, 0, 8)

	cmpStr := func(key, newVal string) {
		if prevVal, ok := prev[key].(string); ok && prevVal != newVal {
			changed = append(changed, key)
		}
	}
	cmpBool := func(key string, newVal bool) {
		if prevVal, ok := prev[key].(bool); ok && prevVal != newVal {
			changed = append(changed, key)
		}
	}

	cmpStr("title", newSchem.Title)
	cmpStr("content", newSchem.Content)
	cmpStr("excerpt", newSchem.Excerpt)
	cmpStr("featured_image", newSchem.FeaturedImage)
	cmpStr("video", newSchem.Video)
	cmpBool("has_dependencies", newSchem.HasDependencies)
	cmpStr("dependencies", newSchem.Dependencies)
	cmpBool("paid", newSchem.Paid)
	cmpStr("schematic_file", newSchem.SchematicFile)

	cmVersionID := ""
	if newSchem.CreatemodVersionID != nil {
		cmVersionID = *newSchem.CreatemodVersionID
	}
	cmpStr("createmod_version", cmVersionID)

	mcVersionID := ""
	if newSchem.MinecraftVersionID != nil {
		mcVersionID = *newSchem.MinecraftVersionID
	}
	cmpStr("minecraft_version", mcVersionID)

	// Check gallery changes
	prevGallery, _ := prev["gallery"].([]string)
	if fmt.Sprintf("%v", prevGallery) != fmt.Sprintf("%v", newSchem.Gallery) {
		changed = append(changed, "gallery")
	}

	return changed
}
