package pages

import (
	"archive/zip"
	"createmod/internal/cache"
	"createmod/internal/store"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// CollectionsDownloadHandler serves a zip file containing all eligible schematics in a collection.
// Eligibility rules (best-effort): skip paid or blacklisted schematics; include only those with a file.
// Association discovery is schema-agnostic: tries join tables (collections_schematics / collection_schematics)
// with fields {collection, schematic}; falls back to a multi-rel field "schematics" on the collection.
// Counters: increments a simple "downloads" int field on the collection if present, and increments
// each included schematic's download counters via countSchematicDownload.
func CollectionsDownloadHandler(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.String(http.StatusBadRequest, "missing collection slug")
		}

		// Resolve collection by slug or id
		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		var collection *core.Record
		if recs, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(recs) > 0 {
			collection = recs[0]
		}
		if collection == nil {
			if rec, err := app.FindRecordById(coll.Id, slug); err == nil {
				collection = rec
			}
		}
		if collection == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}

		// Discover associated schematics
		schematicIDs := make(map[string]struct{})
		// Try join tables first
		for _, jn := range []string{"collections_schematics", "collection_schematics"} {
			if jcoll, jerr := app.FindCollectionByNameOrId(jn); jerr == nil && jcoll != nil {
				if links, lerr := app.FindRecordsByFilter(jcoll.Id, "collection = {:c}", "-created", 2000, 0, dbx.Params{"c": collection.Id}); lerr == nil {
					for _, l := range links {
						sid := l.GetString("schematic")
						if sid != "" {
							schematicIDs[sid] = struct{}{}
						}
					}
				}
			}
		}
		// Fallback to multi-rel field on collection
		for _, sid := range collection.GetStringSlice("schematics") {
			if sid != "" {
				schematicIDs[sid] = struct{}{}
			}
		}

		if len(schematicIDs) == 0 {
			return e.String(http.StatusNotFound, "no schematics in this collection")
		}

		// Load schematic records and gather files
		smColl, err := app.FindCollectionByNameOrId("schematics")
		if err != nil || smColl == nil {
			return e.String(http.StatusInternalServerError, "schematics collection not available")
		}

		type fileItem struct {
			Rec  *core.Record
			Path string
			Name string
		}
		files := make([]fileItem, 0, len(schematicIDs))
		for sid := range schematicIDs {
			rec, err := app.FindRecordById(smColl.Id, sid)
			if err != nil || rec == nil {
				continue
			}
			// Skip paid or blacklisted schematics
			if rec.GetBool("paid") || rec.GetBool("blacklisted") {
				continue
			}
			fname := rec.GetString("schematic_file")
			if fname == "" {
				continue
			}
			// Build local storage path: pb_data/storage/<base>/<file>
			base := rec.BaseFilesPath() // e.g. "schematics/<id>"
			full := filepath.Join("pb_data", "storage", filepath.FromSlash(base), fname)
			// Check file exists before proceeding
			if st, err := os.Stat(full); err != nil || st.IsDir() {
				continue
			}
			// Provide a reasonable filename inside the zip
			display := fname
			if n := strings.TrimSpace(rec.GetString("name")); n != "" {
				// keep extension from fname if present
				if ext := filepath.Ext(fname); ext != "" {
					display = fmt.Sprintf("%s%s", n, ext)
				} else {
					display = n
				}
			}
			files = append(files, fileItem{Rec: rec, Path: full, Name: display})
		}

		if len(files) == 0 {
			return e.String(http.StatusNotFound, "no downloadable schematics in this collection")
		}

		// Increment collection downloads best-effort
		if v := collection.GetString("downloads"); v != "" || true { // attempt regardless; PB will ignore unknown fields
			cur := collection.GetInt("downloads")
			collection.Set("downloads", cur+1)
			if err := app.Save(collection); err != nil {
				app.Logger().Warn("collection: failed to increment downloads", "error", err)
			}
		}

		// Prepare response headers
		zipName := collection.GetString("slug")
		if zipName == "" {
			zipName = collection.GetString("title")
		}
		if zipName == "" {
			zipName = "collection"
		}
		e.Response.Header().Set("Content-Type", "application/zip")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", sanitizeFilename(zipName)))
		e.Response.WriteHeader(http.StatusOK)

		zw := zip.NewWriter(e.Response)
		// Ensure we close the writer; errors after WriteHeader are logged only
		defer zw.Close()

		for _, fi := range files {
			// Increment per-schematic download counters
			countSchematicDownload(app, fi.Rec, e.RealIP(), cacheService)

			fw, err := zw.Create(fi.Name)
			if err != nil {
				app.Logger().Error("zip: create entry failed", "file", fi.Name, "error", err)
				continue
			}
			f, err := os.Open(fi.Path)
			if err != nil {
				app.Logger().Error("zip: open file failed", "path", fi.Path, "error", err)
				continue
			}
			_, err = io.Copy(fw, f)
			_ = f.Close()
			if err != nil {
				app.Logger().Error("zip: copy failed", "file", fi.Name, "error", err)
				continue
			}
		}

		// Returning nil since we already wrote the response
		return nil
	}
}

// sanitizeFilename produces a conservative filename component (very basic).
func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "file"
	}
	// Replace spaces and slashes
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	return s
}
