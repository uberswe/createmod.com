package pages

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"createmod/internal/cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// DownloadHandler redirects to the schematic file and increments a download counter.
// Requires a valid one-time token (?t=) issued by the interstitial page.
func DownloadHandler(app *pocketbase.PocketBase, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		coll, err := app.FindCollectionByNameOrId("schematics")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "schematics collection not available")
		}
		recs, err := app.FindRecordsByFilter(coll.Id, "name = {:name} && deleted = ''", "-created", 1, 0, dbx.Params{"name": name})
		if err != nil || len(recs) == 0 {
			return e.String(http.StatusNotFound, "schematic not found")
		}
		rec := recs[0]

		// Block site download for paid schematics
		if rec.GetBool("paid") {
			return e.String(http.StatusForbidden, "This schematic is paid; please use the external link on the schematic page.")
		}

		// Block download for blacklisted schematics
		if rec.GetBool("blacklisted") {
			return e.String(http.StatusForbidden, "This schematic has been blacklisted and cannot be downloaded.")
		}

		// Increment download counter (best-effort, IP-deduped)
		countSchematicDownload(app, rec, e.RealIP(), cacheService)

		// Determine if there are multiple files associated to this schematic.
		primary := strings.TrimSpace(rec.GetString("schematic_file"))
		base := rec.BaseFilesPath() // e.g. "schematics/<id>"

		// Try to load additional files from schematic_files collection
		multi := make([]string, 0, 8)
		if sfColl, err := app.FindCollectionByNameOrId("schematic_files"); err == nil && sfColl != nil {
			recs2, _ := app.FindRecordsByFilter(sfColl.Id, "schematic = {:s}", "-created", 200, 0, dbx.Params{"s": rec.Id})
			for _, r2 := range recs2 {
				fname := strings.TrimSpace(r2.GetString("file"))
				if fname != "" {
					multi = append(multi, fname)
				}
			}
		}

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
					name := primary
					// prefer pretty name if schema has Title/Name
					if n := strings.TrimSpace(rec.GetString("name")); n != "" {
						if ext := filepath.Ext(primary); ext != "" {
							name = n + ext
						} else {
							name = n
						}
					}
					files = append(files, fileItem{Path: full, Name: name})
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
				zipName := rec.GetString("name")
				if zipName == "" {
					zipName = rec.GetString("title")
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
				return nil
			}
		}

		// Fallback: single file redirect
		if primary == "" {
			return e.String(http.StatusNotFound, "schematic file not found")
		}
		fileURL := fmt.Sprintf("/api/files/%s/%s", rec.BaseFilesPath(), primary)
		return e.Redirect(http.StatusFound, fileURL)
	}
}

// countSchematicDownload increments counters in the "schematic_downloads" collection
// across several periods (total/year/month/week/day), mirroring view counters.
// If the collection is not present, the function logs and returns silently.
// clientIP and cacheService are used for IP-based rate limiting.
func countSchematicDownload(app *pocketbase.PocketBase, schematic *core.Record, clientIP string, cacheService *cache.Service) {
	// IP-based rate limiting: skip if same IP already downloaded this schematic recently
	if clientIP != "" && cacheService != nil {
		ipKey := fmt.Sprintf("dlip:%s:%s", clientIP, schematic.Id)
		if _, already := cacheService.Get(ipKey); already {
			return
		}
		// Mark this IP+schematic combo for 1 hour
		cacheService.SetWithTTL(ipKey, true, 1*time.Hour)
	}

	coll, err := app.FindCollectionByNameOrId("schematic_downloads")
	if err != nil {
		app.Logger().Debug("downloads collection missing", "error", err)
		return
	}
	if coll == nil {
		return
	}

	now := time.Now()
	year, week := now.ISOWeek()
	month := now.Month()
	day := now.Day()

	types := map[int]string{
		4: "total",
		3: fmt.Sprintf("%d", year),
		2: fmt.Sprintf("%d%02d", year, month),
		1: fmt.Sprintf("%d%02d", year, week),
		0: fmt.Sprintf("%d%02d%02d", year, month, day),
	}

	for t, p := range types {
		recs, err := app.FindRecordsByFilter(
			coll.Id,
			"schematic = {:schematic} && type = {:type} && period = {:period}",
			"-created",
			1,
			0,
			dbx.Params{
				"schematic": schematic.Id,
				"type":      t,
				"period":    p,
			},
		)
		if err != nil || len(recs) == 0 {
			if err != nil {
				app.Logger().Debug("downloads query failed", "error", err)
			}
			rec := core.NewRecord(coll)
			rec.Set("schematic", schematic.Id)
			rec.Set("count", 1)
			rec.Set("type", t)
			rec.Set("period", p)
			if err := app.Save(rec); err != nil {
				app.Logger().Error("failed to insert download counter", "error", err)
			}
			if t == 4 {
				cacheService.SetInt(cache.DownloadKey(schematic.Id), 1)
			}
			continue
		}
		cur := recs[0]
		newCount := cur.GetInt("count") + 1
		cur.Set("count", newCount)
		if err := app.Save(cur); err != nil {
			app.Logger().Error("failed to update download counter", "error", err)
		}
		if t == 4 {
			cacheService.SetInt(cache.DownloadKey(schematic.Id), newCount)
		}
	}
}
