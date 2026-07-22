package pages

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// nbtTreeSem caps concurrent NBT decompress+parse work per pod. A single
// request can hold MaxDecompressedSize (100MB) of decompressed NBT plus parse
// overhead, so a crawler fetching several large schematics at once can spike
// the pod past its 4Gi memory limit (OOMKilled wave, 2026-07-22). Requests
// wait briefly for a slot, then get a 429 with Retry-After so well-behaved
// bots back off.
var nbtTreeSem = make(chan struct{}, 2)

// acquireNBTTreeSlot reserves a decompress+parse slot. It returns a release
// function, or writes a 429 and returns false if none frees up in time.
func acquireNBTTreeSlot(e *server.RequestEvent) (func(), bool) {
	select {
	case nbtTreeSem <- struct{}{}:
		return func() { <-nbtTreeSem }, true
	case <-time.After(2 * time.Second):
		e.Response.Header().Set("Retry-After", "5")
		_ = writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "server busy, retry shortly"})
		return nil, false
	}
}

// nbtTreeRespond serves tree/SNBT/search views over raw schematic bytes.
// Shared by the library endpoint and the stateless upload endpoint.
func nbtTreeRespond(e *server.RequestEvent, data []byte) error {
	release, ok := acquireNBTTreeSlot(e)
	if !ok {
		return nil
	}
	defer release()
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

// nbtTreeCacheMaxEntry bounds cached default-page JSON so a crawler sweep
// can't balloon the per-pod cache (worst case ≈ sweep rate × TTL × this).
const nbtTreeCacheMaxEntry = 256 * 1024

// SchematicNBTTreeHandler serves the NBT tree for a library schematic.
// GET /api/schematics/{name}/nbt-tree
func SchematicNBTTreeHandler(appStore *store.Store, cacheService *cache.Service, storageSvc *storage.Service) func(e *server.RequestEvent) error {
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

		// The root page (no path/search/snbt, offset 0) is what every first
		// page load — and every crawler rendering /schematics/{name}/nbt-data
		// — asks for, and it is the same few-hundred-node response every
		// time. It is computed once per schematic version, persisted to S3,
		// and memoized per pod, so bot sweeps never pay the full
		// decompress+parse (up to 100MB in memory per request; caused the
		// 2026-07-22 OOM wave). Deeper navigation (path/offset/snbt/search)
		// stays uncached but is bounded by the nbtTreeSem concurrency cap.
		q := e.Request.URL.Query()
		qInt := func(key string, def int) int {
			if v, err := strconv.Atoi(q.Get(key)); err == nil {
				return v
			}
			return def
		}
		depth := qInt("depth", 2)
		limit := qInt("limit", 200)
		isDefaultPage := strings.TrimSpace(q.Get("q")) == "" && q.Get("snbt") != "1" &&
			q.Get("path") == "" && qInt("offset", 0) == 0

		if isDefaultPage {
			// s.Updated in the keys makes them self-invalidating: replacing
			// the file bumps Updated, so a new key is computed and the old
			// objects just go stale (a few KB each, harmless orphans).
			memKey := fmt.Sprintf("nbt_tree_p1_%s_%d_%d_%d", s.ID, s.Updated.Unix(), depth, limit)
			s3Key := fmt.Sprintf("_nbt_tree/%s/%d_%d_%d.json", s.ID, s.Updated.Unix(), depth, limit)
			serve := func(body []byte) error {
				// Shared-cache friendly (unlike setPublicCacheControl, which
				// sets s-maxage=0): the payload is public, identical for
				// every viewer, and version-keyed server-side, so the CDN
				// may hold it for hours.
				e.Response.Header().Set("Cache-Control", "public, max-age=3600, s-maxage=21600, stale-while-revalidate=86400")
				e.Response.Header().Set("Content-Type", "application/json")
				e.Response.WriteHeader(http.StatusOK)
				_, _ = e.Response.Write(body)
				return nil
			}
			if cacheService != nil {
				if v, ok := cacheService.Get(memKey); ok {
					if body, ok2 := v.([]byte); ok2 {
						return serve(body)
					}
				}
			}
			if r, err := storageSvc.DownloadRaw(e.Request.Context(), s3Key); err == nil {
				body, rerr := io.ReadAll(io.LimitReader(r, nbtTreeCacheMaxEntry+1))
				_ = r.Close()
				if rerr == nil && len(body) > 0 && len(body) <= nbtTreeCacheMaxEntry {
					if cacheService != nil {
						cacheService.SetWithTTL(memKey, body, 10*time.Minute)
					}
					return serve(body)
				}
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

			release, ok := acquireNBTTreeSlot(e)
			if !ok {
				return nil
			}
			defer release()
			raw, err := schematic.DecompressForTree(data)
			if err != nil {
				return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
			}
			page, err := schematic.NBTTreePage(raw, "", depth, 0, limit)
			if err != nil {
				return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
			}
			body, err := json.Marshal(page)
			if err != nil {
				return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to encode tree"})
			}
			if len(body) <= nbtTreeCacheMaxEntry {
				if cacheService != nil {
					cacheService.SetWithTTL(memKey, body, 10*time.Minute)
				}
				if err := storageSvc.UploadRawBytes(e.Request.Context(), s3Key, body, "application/json"); err != nil {
					slog.Warn("nbt-tree: failed to persist root page", "key", s3Key, "error", err)
				}
			}
			return serve(body)
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
