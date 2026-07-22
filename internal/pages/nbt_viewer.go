package pages

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// nbtTreeRespond serves tree/SNBT/search views over raw schematic bytes.
// Shared by the library endpoint and the stateless upload endpoint.
func nbtTreeRespond(e *server.RequestEvent, data []byte) error {
	raw, err := schematic.DecompressForTree(data)
	if err != nil {
		return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
	}
	q := e.Request.URL.Query()

	if search := strings.TrimSpace(q.Get("q")); search != "" {
		hits, err := schematic.NBTTreeSearch(raw, search, 200)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}
		return writeJSON(e, http.StatusOK, map[string]interface{}{"results": hits})
	}

	path := q.Get("path")
	if q.Get("snbt") == "1" {
		snbt, err := schematic.NBTNodeSNBT(raw, path, 0)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}
		return writeJSON(e, http.StatusOK, map[string]string{"snbt": snbt})
	}

	atoi := func(key string, def int) int {
		if v, err := strconv.Atoi(q.Get(key)); err == nil {
			return v
		}
		return def
	}
	page, err := schematic.NBTTreePage(raw, path, atoi("depth", 2), atoi("offset", 0), atoi("limit", 200))
	if err != nil {
		return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
	}
	return writeJSON(e, http.StatusOK, page)
}

// SchematicNBTTreeHandler serves the NBT tree for a library schematic.
// GET /api/schematics/{name}/nbt-tree
func SchematicNBTTreeHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if storageSvc == nil {
			return writeJSON(e, http.StatusServiceUnavailable, map[string]string{"error": "file storage unavailable"})
		}
		name := e.Request.PathValue("name")
		s, err := appStore.Schematics.GetByName(e.Request.Context(), name)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic not found"})
		}
		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic has no file"})
		}
		reader, err := storageSvc.Download(e.Request.Context(), storage.CollectionPrefix("schematics"), s.ID, primary)
		if err != nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "schematic file not found"})
		}
		defer reader.Close()
		data, err := io.ReadAll(io.LimitReader(reader, maxUploadSize+1))
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to read schematic"})
		}
		// Tree responses vary only with the stored file; cache briefly.
		setPublicCacheControl(e, 300)
		return nbtTreeRespond(e, data)
	}
}

// NBTTreeUploadHandler is the stateless variant: the client re-sends the
// file with each expansion request; nothing is stored server-side.
// POST /api/nbt-tree (multipart: file, plus the same query params)
func NBTTreeUploadHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(maxUploadSize + 1<<20); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid form"})
		}
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing file"})
		}
		defer file.Close()
		if header.Size > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil || int64(len(data)) > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}
		return nbtTreeRespond(e, data)
	}
}

var nbtViewerTemplates = append([]string{
	"./template/nbt_viewer.html",
}, commonTemplates...)

// NBTViewerToolHandler renders /tools/nbt-viewer.
func NBTViewerToolHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		setPublicCacheControl(e, 600)
		d := safetyPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "NBT Viewer Online - Inspect Minecraft NBT and Schematic Files")
		d.Description = i18n.T(d.Language, "Free online NBT viewer: open .nbt, .schem, .litematic and .schematic files in a browsable tree with SNBT view, key search and copy-path. Files stay in your browser session, never stored.")
		d.Slug = "/tools/nbt-viewer"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "NBT Viewer"))
		html, err := registry.LoadFiles(nbtViewerTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

var nbtDataTemplates = append([]string{
	"./template/nbt_data.html",
}, commonTemplates...)

type nbtDataPageData struct {
	DefaultData
	SourceTitle string
	SourceName  string
}

// SchematicNBTDataHandler renders /schematics/{name}/nbt-data: the NBT tree
// for a published schematic's file, with a back link to the schematic.
func SchematicNBTDataHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		s, err := appStore.Schematics.GetByName(e.Request.Context(), name)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return FourOhFourHandler(registry, appStore)(e)
		}
		d := nbtDataPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = fmt.Sprintf(i18n.T(d.Language, "NBT Data: %s"), s.Title)
		d.Description = i18n.T(d.Language, "Browse the raw NBT structure of this schematic.")
		d.Slug = "/schematics/" + name + "/nbt-data"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Schematics"), "/schematics", s.Title, "/schematics/"+name, i18n.T(d.Language, "NBT Data"))
		d.SourceTitle = s.Title
		d.SourceName = s.Name
		html, err := registry.LoadFiles(nbtDataTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
