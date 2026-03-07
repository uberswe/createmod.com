package pages

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/store"

	"createmod/internal/server"
)

// DownloadHandler redirects to the schematic file and increments a download counter.
// Requires a valid one-time token (?t=) issued by the interstitial page.
func DownloadHandler(cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		if name == "" {
			return e.String(http.StatusBadRequest, "missing name")
		}
		// validate one-time token
		token := e.Request.URL.Query().Get("t")
		if token == "" {
			return e.String(http.StatusForbidden, "missing download token; please open the download page again")
		}
		storedName, ok := cacheService.GetString("dl:" + token)
		if !ok || storedName == "" || storedName != name {
			return e.String(http.StatusForbidden, "invalid or expired download token; please open the download page again")
		}
		// consume token
		cacheService.Delete("dl:" + token)

		s, err := appStore.Schematics.GetByName(context.Background(), name)
		if err != nil || s == nil || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return e.String(http.StatusNotFound, "schematic not found")
		}

		// Block site download for paid schematics
		if s.Paid {
			return e.String(http.StatusForbidden, "This schematic is paid; please use the external link on the schematic page.")
		}

		// Block download for blacklisted schematics
		if s.Blacklisted {
			return e.String(http.StatusForbidden, "This schematic has been blacklisted and cannot be downloaded.")
		}

		// Increment download counter (best-effort, IP-deduped)
		countSchematicDownloadStore(appStore, s.ID, e.RealIP(), cacheService)

		// Determine if there are multiple files associated to this schematic.
		primary := strings.TrimSpace(s.SchematicFile)
		base := "schematics/" + s.ID

		// TODO: schematic_files collection not yet in store; multi-file zip disabled for now
		multi := make([]string, 0)

		// If we have additional files and at least one existing on disk (including primary), stream a zip
		// Otherwise, fallback to single file redirect as before.
		if len(multi) > 0 {
			type fileItem struct {
				Path string
				Name string
			}
			files := make([]fileItem, 0, len(multi)+1)
			seen := map[string]struct{}{}
			// include primary first if present
			if primary != "" {
				full := filepath.Join("pb_data", "storage", filepath.FromSlash(base), primary)
				if st, err := os.Stat(full); err == nil && !st.IsDir() {
					fname := primary
					if n := strings.TrimSpace(s.Name); n != "" {
						if ext := filepath.Ext(primary); ext != "" {
							fname = n + ext
						} else {
							fname = n
						}
					}
					files = append(files, fileItem{Path: full, Name: fname})
					seen[primary] = struct{}{}
				}
			}
			for _, fname := range multi {
				if _, ok := seen[fname]; ok {
					continue
				}
				full := filepath.Join("pb_data", "storage", filepath.FromSlash(base), fname)
				if st, err := os.Stat(full); err == nil && !st.IsDir() {
					files = append(files, fileItem{Path: full, Name: fname})
					seen[fname] = struct{}{}
				}
			}
			if len(files) > 0 {
				zipName := s.Name
				if zipName == "" {
					zipName = s.Title
				}
				if zipName == "" {
					zipName = "schematic"
				}
				e.Response.Header().Set("Content-Type", "application/zip")
				e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", sanitizeFilename(zipName)))
				e.Response.WriteHeader(http.StatusOK)
				zw := zip.NewWriter(e.Response)
				defer zw.Close()
				for _, fi := range files {
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
				return nil
			}
		}

		// Fallback: single file redirect
		if primary == "" {
			return e.String(http.StatusNotFound, "schematic file not found")
		}
		fileURL := fmt.Sprintf("/api/files/%s/%s", base, primary)
		return e.Redirect(http.StatusFound, fileURL)
	}
}

// countSchematicDownloadStore increments download counters via the PostgreSQL store.
// clientIP and cacheService are used for IP-based rate limiting.
func countSchematicDownloadStore(appStore *store.Store, schematicID string, clientIP string, cacheService *cache.Service) {
	// IP-based rate limiting: skip if same IP already downloaded this schematic recently
	if clientIP != "" && cacheService != nil {
		ipKey := fmt.Sprintf("dlip:%s:%s", clientIP, schematicID)
		if _, already := cacheService.Get(ipKey); already {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
	}

	if err := appStore.ViewRatings.RecordDownload(context.Background(), schematicID, nil); err != nil {
		return
	}
	// Update cache with new total
	if total, err := appStore.ViewRatings.GetDownloadCount(context.Background(), schematicID); err == nil {
		cacheService.SetInt(cache.DownloadKey(schematicID), total)
	}
}
