package pages

import (
	"archive/zip"
	"context"
	"createmod/internal/cache"
	"createmod/internal/store"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"createmod/internal/server"
)

// CollectionsDownloadHandler serves a zip file containing all eligible schematics in a collection.
// Eligibility rules (best-effort): skip paid or blacklisted schematics; include only those with a file.
// Counters: increments a simple "downloads" int field on the collection if present, and increments
// each included schematic's download counters via countSchematicDownloadStore.
func CollectionsDownloadHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.String(http.StatusBadRequest, "missing collection slug")
		}

		ctx := context.Background()

		// Resolve collection by slug or id
		collection, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || collection == nil {
			// Try by ID as fallback
			collection, err = appStore.Collections.GetByID(ctx, slug)
			if err != nil || collection == nil {
				return e.String(http.StatusNotFound, "collection not found")
			}
		}

		// Discover associated schematics
		schematicIDs, err := appStore.Collections.GetSchematicIDs(ctx, collection.ID)
		if err != nil || len(schematicIDs) == 0 {
			return e.String(http.StatusNotFound, "no schematics in this collection")
		}

		// Load schematic records and gather files
		schematics, err := appStore.Schematics.ListByIDs(ctx, schematicIDs)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to load schematics")
		}

		type fileItem struct {
			SchematicID string
			Path        string
			Name        string
		}
		files := make([]fileItem, 0, len(schematics))
		for _, s := range schematics {
			// Skip paid or blacklisted schematics
			if s.Paid || s.Blacklisted {
				continue
			}
			fname := s.SchematicFile
			if fname == "" {
				continue
			}
			// Build local storage path: pb_data/storage/schematics/<id>/<file>
			base := "schematics/" + s.ID
			full := filepath.Join("pb_data", "storage", filepath.FromSlash(base), fname)
			// Check file exists before proceeding
			if st, err := os.Stat(full); err != nil || st.IsDir() {
				continue
			}
			// Provide a reasonable filename inside the zip
			display := fname
			if n := strings.TrimSpace(s.Name); n != "" {
				if ext := filepath.Ext(fname); ext != "" {
					display = fmt.Sprintf("%s%s", n, ext)
				} else {
					display = n
				}
			}
			files = append(files, fileItem{SchematicID: s.ID, Path: full, Name: display})
		}

		if len(files) == 0 {
			return e.String(http.StatusNotFound, "no downloadable schematics in this collection")
		}

		// Increment collection downloads best-effort
		_ = appStore.Collections.IncrementViews(ctx, collection.ID)

		// Prepare response headers
		zipName := collection.Slug
		if zipName == "" {
			zipName = collection.Title
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
			countSchematicDownloadStore(appStore, fi.SchematicID, e.RealIP(), cacheService)

			fw, err := zw.Create(fi.Name)
			if err != nil {
				slog.Error("zip: create entry failed", "file", fi.Name, "error", err)
				continue
			}
			f, err := os.Open(fi.Path)
			if err != nil {
				slog.Error("zip: open file failed", "path", fi.Path, "error", err)
				continue
			}
			_, err = io.Copy(fw, f)
			_ = f.Close()
			if err != nil {
				slog.Error("zip: copy failed", "file", fi.Name, "error", err)
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
