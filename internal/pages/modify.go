package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/nbtparser"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

var modifyTemplates = append([]string{
	"./template/modify.html",
}, commonTemplates...)

var uploadModifyTemplates = append([]string{
	"./template/upload_modify.html",
}, commonTemplates...)

// ModifyData holds template data for the schematic modify page.
type ModifyData struct {
	DefaultData
	Schematic        store.Schematic
	Materials        []nbtparser.Material
	Variations       []*store.SchematicVariation
	PublicVariations  []*store.SchematicVariation
	PreloadedJSON    string // Pre-filled replacements JSON (from ?v=variationID)
	SchematicID      string
}

// ModifyHandler renders the block replacement page for a schematic.
func ModifyHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		slug := chi.URLParam(e.Request, "name")
		if slug == "" {
			return e.NotFoundError("Not found", nil)
		}

		ctx := e.Request.Context()
		s, err := appStore.Schematics.GetByName(ctx, slug)
		if err != nil || s == nil {
			return e.NotFoundError("Schematic not found", nil)
		}

		// Must be published (moderated) or the user is the owner
		userID := authenticatedUserID(e)
		isOwner := s.AuthorID == userID
		isPublished := s.Deleted == nil && s.Moderated
		if !isPublished && !isOwner {
			return e.NotFoundError("Schematic not found", nil)
		}

		// Extract materials from NBT
		var materials []nbtparser.Material
		if storageService != nil && s.SchematicFile != "" {
			reader, dlErr := storageService.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, s.SchematicFile)
			if dlErr == nil {
				data, readErr := io.ReadAll(reader)
				reader.Close()
				if readErr == nil {
					mats, matErr := nbtparser.ExtractMaterials(data)
					if matErr == nil {
						materials = mats
					}
				}
			}
		}

		// If materials are not available from NBT, try from stored JSON
		if len(materials) == 0 && len(s.Materials) > 0 {
			var storedMats []nbtparser.Material
			if json.Unmarshal(s.Materials, &storedMats) == nil {
				materials = storedMats
			}
		}

		d := ModifyData{
			Schematic:   *s,
			Materials:   materials,
			SchematicID: s.ID,
		}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "modify_blocks") + " - " + s.Title
		d.Description = i18n.T(d.Language, "modify_blocks_description")
		d.Breadcrumbs = NewBreadcrumbs(d.Language,
			i18n.T(d.Language, "Schematics"), "/schematics",
			s.Title, "/schematics/"+s.Name,
			i18n.T(d.Language, "modify_blocks"),
		)

		// Load user's saved variations
		if appStore.SchematicVariations != nil {
			vars, varErr := appStore.SchematicVariations.ListBySchematicAndUser(ctx, s.ID, userID)
			if varErr == nil {
				d.Variations = vars
			}
			pubVars, pubErr := appStore.SchematicVariations.ListPublicBySchematic(ctx, s.ID)
			if pubErr == nil {
				d.PublicVariations = pubVars
			}
		}

		// Pre-fill from variation if ?v= param present
		if vid := e.Request.URL.Query().Get("v"); vid != "" && appStore.SchematicVariations != nil {
			v, vErr := appStore.SchematicVariations.GetByID(ctx, vid)
			if vErr == nil && v != nil && v.SchematicID == s.ID {
				if v.UserID == userID || v.IsPublic {
					d.PreloadedJSON = string(v.Replacements)
				}
			}
		}

		html, renderErr := registry.LoadFiles(modifyTemplates...).Render(d)
		if renderErr != nil {
			return renderErr
		}
		return e.HTML(http.StatusOK, html)
	}
}

// modifyRequest is the JSON body for download/preview endpoints.
type modifyRequest struct {
	Replacements []nbtparser.ReplaceBlock `json:"replacements"`
}

// ModifyDownloadHandler handles POST /api/schematics/{id}/modify/download.
func ModifyDownloadHandler(appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		schematicID := chi.URLParam(e.Request, "id")
		if schematicID == "" {
			return e.BadRequestError("missing schematic ID", nil)
		}

		ctx := e.Request.Context()
		s, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || s == nil {
			return e.NotFoundError("Schematic not found", nil)
		}

		if storageService == nil || s.SchematicFile == "" {
			return e.BadRequestError("Schematic file not available", nil)
		}

		var req modifyRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.BadRequestError("invalid JSON body", nil)
		}

		if len(req.Replacements) == 0 {
			return e.BadRequestError("no replacements provided", nil)
		}

		if len(req.Replacements) > 1000 {
			return e.BadRequestError("too many replacements (max 1000)", nil)
		}

		for _, r := range req.Replacements {
			if !nbtparser.ValidateBlockID(r.OriginalID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.OriginalID), nil)
			}
			if r.ReplacementID != "" && !nbtparser.ValidateBlockID(r.ReplacementID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.ReplacementID), nil)
			}
		}

		reader, err := storageService.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, s.SchematicFile)
		if err != nil {
			slog.Error("failed to download schematic for modify", "error", err)
			return e.InternalServerError("failed to download schematic", nil)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return e.InternalServerError("failed to read schematic", nil)
		}

		modified, err := nbtparser.ReplacePalette(data, req.Replacements)
		if err != nil {
			slog.Error("failed to apply palette replacements", "error", err)
			return e.InternalServerError("failed to modify schematic: "+err.Error(), nil)
		}

		filename := s.Name + "-modified.nbt"
		filename = sanitizeContentDispositionFilename(filename)

		e.Response.Header().Set("Content-Type", "application/octet-stream")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Header().Set("Content-Length", fmt.Sprintf("%d", len(modified)))
		e.Response.WriteHeader(http.StatusOK)
		_, _ = e.Response.Write(modified)
		return nil
	}
}

// ModifyPreviewHandler handles POST /api/schematics/{id}/modify/preview-url.
func ModifyPreviewHandler(appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		schematicID := chi.URLParam(e.Request, "id")
		if schematicID == "" {
			return e.BadRequestError("missing schematic ID", nil)
		}

		ctx := e.Request.Context()
		s, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || s == nil {
			return e.NotFoundError("Schematic not found", nil)
		}

		if storageService == nil || s.SchematicFile == "" {
			return e.BadRequestError("Schematic file not available", nil)
		}

		var req modifyRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.BadRequestError("invalid JSON body", nil)
		}

		if len(req.Replacements) == 0 {
			return e.BadRequestError("no replacements provided", nil)
		}

		if len(req.Replacements) > 1000 {
			return e.BadRequestError("too many replacements (max 1000)", nil)
		}

		reader, err := storageService.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, s.SchematicFile)
		if err != nil {
			return e.InternalServerError("failed to download schematic", nil)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return e.InternalServerError("failed to read schematic", nil)
		}

		modified, err := nbtparser.ReplacePalette(data, req.Replacements)
		if err != nil {
			return e.InternalServerError("failed to modify schematic: "+err.Error(), nil)
		}

		tempKey := fmt.Sprintf("temp/variations/%s-%s.nbt", schematicID, generateTempID())
		if err := storageService.UploadRawBytes(ctx, tempKey, modified, "application/octet-stream"); err != nil {
			return e.InternalServerError("failed to upload preview", nil)
		}

		scheme := "http"
		if e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		fileURL := fmt.Sprintf("%s://%s/api/files/_raw/%s", scheme, e.Request.Host, url.PathEscape(tempKey))
		bloxelizerURL := "https://bloxelizer.com/viewer?url=" + url.QueryEscape(fileURL)

		return e.JSON(http.StatusOK, map[string]string{
			"url": bloxelizerURL,
		})
	}
}

// generateTempID generates a short random hex ID for temp file naming.
func generateTempID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateVariationHandler handles POST /api/schematics/{id}/variations.
func CreateVariationHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		schematicID := chi.URLParam(e.Request, "id")
		if schematicID == "" {
			return e.BadRequestError("missing schematic ID", nil)
		}

		ctx := e.Request.Context()
		s, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || s == nil {
			return e.NotFoundError("Schematic not found", nil)
		}

		userID := authenticatedUserID(e)

		var req struct {
			Name         string                  `json:"name"`
			Replacements []nbtparser.ReplaceBlock `json:"replacements"`
			IsPublic     bool                     `json:"is_public"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.BadRequestError("invalid JSON body", nil)
		}

		if len(req.Name) > 100 {
			return e.BadRequestError("name too long (max 100 characters)", nil)
		}
		if len(req.Replacements) == 0 {
			return e.BadRequestError("no replacements provided", nil)
		}
		if len(req.Replacements) > 1000 {
			return e.BadRequestError("too many replacements (max 1000)", nil)
		}
		for _, r := range req.Replacements {
			if !nbtparser.ValidateBlockID(r.OriginalID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.OriginalID), nil)
			}
			if r.ReplacementID != "" && !nbtparser.ValidateBlockID(r.ReplacementID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.ReplacementID), nil)
			}
		}

		if appStore.SchematicVariations != nil {
			count, cErr := appStore.SchematicVariations.CountBySchematicAndUser(ctx, schematicID, userID)
			if cErr == nil && count >= 50 {
				return e.BadRequestError("maximum 50 variations per schematic", nil)
			}
		}

		replacementsJSON, err := json.Marshal(req.Replacements)
		if err != nil {
			return e.InternalServerError("failed to encode replacements", nil)
		}

		v := &store.SchematicVariation{
			SchematicID:  schematicID,
			UserID:       userID,
			Name:         req.Name,
			Replacements: replacementsJSON,
			IsPublic:     req.IsPublic,
		}
		if err := appStore.SchematicVariations.Create(ctx, v); err != nil {
			slog.Error("failed to create variation", "error", err)
			return e.InternalServerError("failed to save variation", nil)
		}

		return e.JSON(http.StatusCreated, v)
	}
}

// DeleteVariationHandler handles DELETE /api/schematics/{id}/variations/{variationID}.
func DeleteVariationHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		variationID := chi.URLParam(e.Request, "variationID")
		if variationID == "" {
			return e.BadRequestError("missing variation ID", nil)
		}

		ctx := e.Request.Context()
		v, err := appStore.SchematicVariations.GetByID(ctx, variationID)
		if err != nil || v == nil {
			return e.NotFoundError("Variation not found", nil)
		}

		userID := authenticatedUserID(e)
		if v.UserID != userID {
			return e.ForbiddenError("not authorized", nil)
		}

		if err := appStore.SchematicVariations.Delete(ctx, variationID); err != nil {
			return e.InternalServerError("failed to delete variation", nil)
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// ListVariationsHandler handles GET /api/schematics/{id}/variations.
func ListVariationsHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		schematicID := chi.URLParam(e.Request, "id")
		if schematicID == "" {
			return e.BadRequestError("missing schematic ID", nil)
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		var userVars []*store.SchematicVariation
		var publicVars []*store.SchematicVariation

		if appStore.SchematicVariations != nil {
			uv, err := appStore.SchematicVariations.ListBySchematicAndUser(ctx, schematicID, userID)
			if err == nil {
				userVars = uv
			}
			pv, err := appStore.SchematicVariations.ListPublicBySchematic(ctx, schematicID)
			if err == nil {
				publicVars = pv
			}
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"user_variations":   userVars,
			"public_variations": publicVars,
		})
	}
}

// UploadModifyData holds template data for the private upload modify page.
type UploadModifyData struct {
	DefaultData
	Token     string
	Filename  string
	Materials []nbtparser.Material
}

// UploadModifyHandler renders the block replacement page for a private upload.
// GET /u/{token}/modify
func UploadModifyHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.BadRequestError("missing token", nil)
		}

		ctx := e.Request.Context()
		entry, err := appStore.TempUploads.GetByToken(ctx, token)
		if err != nil {
			return e.NotFoundError("invalid or expired token", nil)
		}

		// Must be the owner or unclaimed
		userID := authenticatedUserID(e)
		if entry.UploadedBy != "" && entry.UploadedBy != userID {
			return e.ForbiddenError("not authorized", nil)
		}

		// Extract materials from NBT
		var materials []nbtparser.Material
		if storageService != nil && entry.NbtS3Key != "" {
			reader, dlErr := storageService.DownloadRaw(ctx, entry.NbtS3Key)
			if dlErr == nil {
				data, readErr := io.ReadAll(reader)
				reader.Close()
				if readErr == nil {
					mats, matErr := nbtparser.ExtractMaterials(data)
					if matErr == nil {
						materials = mats
					}
				}
			}
		}

		// Fallback to stored materials JSON
		if len(materials) == 0 && len(entry.Materials) > 0 {
			var storedMats []nbtparser.Material
			if json.Unmarshal(entry.Materials, &storedMats) == nil {
				materials = storedMats
			}
		}

		d := UploadModifyData{
			Token:     token,
			Filename:  entry.Filename,
			Materials: materials,
		}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "modify_blocks") + " - " + entry.Filename
		d.Description = i18n.T(d.Language, "modify_blocks_description")
		d.Breadcrumbs = NewBreadcrumbs(d.Language,
			i18n.T(d.Language, "Upload"), "/upload",
			i18n.T(d.Language, "Preview"), "/u/"+token,
			i18n.T(d.Language, "modify_blocks"),
		)

		html, renderErr := registry.LoadFiles(uploadModifyTemplates...).Render(d)
		if renderErr != nil {
			return renderErr
		}
		return e.HTML(http.StatusOK, html)
	}
}

// UploadModifyDownloadHandler handles POST /u/{token}/modify/download.
func UploadModifyDownloadHandler(appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.BadRequestError("missing token", nil)
		}

		ctx := e.Request.Context()
		entry, err := appStore.TempUploads.GetByToken(ctx, token)
		if err != nil {
			return e.NotFoundError("invalid or expired token", nil)
		}

		userID := authenticatedUserID(e)
		if entry.UploadedBy != "" && entry.UploadedBy != userID {
			return e.ForbiddenError("not authorized", nil)
		}

		if storageService == nil || entry.NbtS3Key == "" {
			return e.BadRequestError("file not available", nil)
		}

		var req modifyRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.BadRequestError("invalid JSON body", nil)
		}

		if len(req.Replacements) == 0 {
			return e.BadRequestError("no replacements provided", nil)
		}
		if len(req.Replacements) > 1000 {
			return e.BadRequestError("too many replacements (max 1000)", nil)
		}

		for _, r := range req.Replacements {
			if !nbtparser.ValidateBlockID(r.OriginalID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.OriginalID), nil)
			}
			if r.ReplacementID != "" && !nbtparser.ValidateBlockID(r.ReplacementID) {
				return e.BadRequestError(fmt.Sprintf("invalid block ID: %s", r.ReplacementID), nil)
			}
		}

		reader, err := storageService.DownloadRaw(ctx, entry.NbtS3Key)
		if err != nil {
			slog.Error("failed to download private upload for modify", "error", err)
			return e.InternalServerError("failed to download schematic", nil)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return e.InternalServerError("failed to read schematic", nil)
		}

		modified, err := nbtparser.ReplacePalette(data, req.Replacements)
		if err != nil {
			slog.Error("failed to apply palette replacements", "error", err)
			return e.InternalServerError("failed to modify schematic: "+err.Error(), nil)
		}

		filename := strings.TrimSuffix(entry.Filename, ".nbt") + "-modified.nbt"
		filename = sanitizeContentDispositionFilename(filename)

		e.Response.Header().Set("Content-Type", "application/octet-stream")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		e.Response.Header().Set("Content-Length", fmt.Sprintf("%d", len(modified)))
		e.Response.WriteHeader(http.StatusOK)
		_, _ = e.Response.Write(modified)
		return nil
	}
}

// UploadModifyPreviewHandler handles POST /u/{token}/modify/preview-url.
func UploadModifyPreviewHandler(appStore *store.Store, storageService *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ok, err := requireAuth(e)
		if !ok {
			return err
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.BadRequestError("missing token", nil)
		}

		ctx := e.Request.Context()
		entry, err := appStore.TempUploads.GetByToken(ctx, token)
		if err != nil {
			return e.NotFoundError("invalid or expired token", nil)
		}

		userID := authenticatedUserID(e)
		if entry.UploadedBy != "" && entry.UploadedBy != userID {
			return e.ForbiddenError("not authorized", nil)
		}

		if storageService == nil || entry.NbtS3Key == "" {
			return e.BadRequestError("file not available", nil)
		}

		var req modifyRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.BadRequestError("invalid JSON body", nil)
		}

		if len(req.Replacements) == 0 {
			return e.BadRequestError("no replacements provided", nil)
		}
		if len(req.Replacements) > 1000 {
			return e.BadRequestError("too many replacements (max 1000)", nil)
		}

		reader, err := storageService.DownloadRaw(ctx, entry.NbtS3Key)
		if err != nil {
			return e.InternalServerError("failed to download schematic", nil)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return e.InternalServerError("failed to read schematic", nil)
		}

		modified, err := nbtparser.ReplacePalette(data, req.Replacements)
		if err != nil {
			return e.InternalServerError("failed to modify schematic: "+err.Error(), nil)
		}

		tempKey := fmt.Sprintf("temp/variations/%s-%s.nbt", token, generateTempID())
		if err := storageService.UploadRawBytes(ctx, tempKey, modified, "application/octet-stream"); err != nil {
			return e.InternalServerError("failed to upload preview", nil)
		}

		scheme := "http"
		if e.Request.TLS != nil || strings.EqualFold(e.Request.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		fileURL := fmt.Sprintf("%s://%s/api/files/_raw/%s", scheme, e.Request.Host, url.PathEscape(tempKey))
		bloxelizerURL := "https://bloxelizer.com/viewer?url=" + url.QueryEscape(fileURL)

		return e.JSON(http.StatusOK, map[string]string{
			"url": bloxelizerURL,
		})
	}
}
