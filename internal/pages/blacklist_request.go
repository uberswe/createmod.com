package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/nbtparser"
	"createmod/internal/server"
	"createmod/internal/store"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var blacklistRequestTemplates = append([]string{
	"./template/blacklist_request.html",
}, commonTemplates...)

type BlacklistRequestData struct {
	DefaultData
	Hashes []store.NBTHash
}

// BlacklistRequestHandler renders the blacklist page where authenticated users
// can upload .nbt files whose hashes will be stored to prevent future uploads.
func BlacklistRequestHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := BlacklistRequestData{}
		d.Populate(e)
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Blacklist Schematics"))
		d.Title = i18n.T(d.Language, "Blacklist Schematics")
		d.Description = i18n.T(d.Language, "page.blacklist.description")
		d.Slug = "/settings/blacklist"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		hashes, _ := appStore.NBTHashes.ListByUser(context.Background(), d.UserID)
		d.Hashes = hashes

		html, err := registry.LoadFiles(blacklistRequestTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// BlacklistUploadHandler handles POST /settings/blacklist/upload.
// Accepts multipart .nbt files, validates them, computes SHA256 hashes,
// and stores the hashes in nbt_hashes with schematic_id=nil.
func BlacklistUploadHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)

		const maxBlacklistUpload = 10 << 20 // 10 MB per file
		_ = e.Request.ParseMultipartForm(maxBlacklistUpload * 10)

		if e.Request.MultipartForm == nil || len(e.Request.MultipartForm.File["files"]) == 0 {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "no files uploaded"})
		}
		files := e.Request.MultipartForm.File["files"]

		type result struct {
			Filename string `json:"filename"`
			Hash     string `json:"hash"`
			Status   string `json:"status"`
			Error    string `json:"error,omitempty"`
		}
		results := make([]result, 0, len(files))

		for _, fh := range files {
			r := result{Filename: fh.Filename}

			// Validate .nbt extension
			if !strings.HasSuffix(strings.ToLower(fh.Filename), ".nbt") {
				r.Status = "error"
				r.Error = "not an .nbt file"
				results = append(results, r)
				continue
			}

			f, err := fh.Open()
			if err != nil {
				r.Status = "error"
				r.Error = "failed to open file"
				results = append(results, r)
				continue
			}
			data, err := io.ReadAll(io.LimitReader(f, maxBlacklistUpload+1))
			f.Close()
			if err != nil || int64(len(data)) > maxBlacklistUpload {
				r.Status = "error"
				r.Error = "file too large or unreadable"
				results = append(results, r)
				continue
			}

			// Validate NBT
			if ok, reason := nbtparser.Validate(data); !ok {
				r.Status = "error"
				r.Error = "invalid NBT file"
				if reason != "" {
					r.Error += ": " + reason
				}
				results = append(results, r)
				continue
			}

			// Compute SHA256
			sum := sha256.Sum256(data)
			checksum := hex.EncodeToString(sum[:])
			r.Hash = checksum

			// Check if already blacklisted
			if blacklisted, err := appStore.NBTHashes.IsBlacklisted(context.Background(), checksum); err == nil && blacklisted {
				r.Status = "exists"
				r.Error = "already blacklisted"
				results = append(results, r)
				continue
			}

			// Also check if this hash belongs to a published schematic
			if existingID, err := appStore.Schematics.GetByChecksum(context.Background(), checksum); err == nil && existingID != "" {
				r.Status = "exists"
				r.Error = "this hash is already associated with a published schematic"
				results = append(results, r)
				continue
			}

			// Insert into nbt_hashes
			idBuf := make([]byte, 8)
			_, _ = rand.Read(idBuf)
			id := hex.EncodeToString(idBuf)[:15]
			err = appStore.NBTHashes.Create(context.Background(), &store.NBTHash{
				ID:         id,
				Hash:       checksum,
				UploadedBy: &userID,
			})
			if err != nil {
				r.Status = "error"
				r.Error = "failed to save hash"
				results = append(results, r)
				continue
			}

			r.Status = "ok"
			results = append(results, r)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{"results": results})
	}
}

// BlacklistDeleteHandler handles DELETE /settings/blacklist/{id}.
// Removes a blacklisted hash owned by the authenticated user.
func BlacklistDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		id := chi.URLParam(e.Request, "id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing id"})
		}
		userID := authenticatedUserID(e)
		if err := appStore.NBTHashes.Delete(context.Background(), id, userID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
