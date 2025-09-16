package pages

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"createmod/internal/cache"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// APIKeyCreateHandler handles POST /settings/api-keys/new
// Auth required. Generates a random API key, stores its sha256 hash and shows
// the plaintext once via a temporary cache entry.
func APIKeyCreateHandler(app *pocketbase.PocketBase, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		label := strings.TrimSpace(e.Request.FormValue("label"))

		coll, err := app.FindCollectionByNameOrId("api_keys")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "api keys collection not available")
		}

		// Generate 32 random bytes and hex encode (64 chars)
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return e.String(http.StatusInternalServerError, "failed to generate key")
		}
		plaintext := hex.EncodeToString(buf)
		sum := sha256.Sum256([]byte(plaintext))
		hash := hex.EncodeToString(sum[:])
		last8 := ""
		if len(plaintext) >= 8 {
			last8 = plaintext[len(plaintext)-8:]
		}

		rec := core.NewRecord(coll)
		rec.Set("user", e.Auth.Id)
		rec.Set("key_hash", hash)
		if label != "" {
			rec.Set("label", label)
		}
		if last8 != "" {
			rec.Set("last8", last8)
		}
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save api key")
		}

		// Cache plaintext for one-time display on /settings
		cacheService.SetWithTTL("apikey:new:"+e.Auth.Id, plaintext, 2*time.Minute)

		dest := "/settings?api_key=created"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// APIKeyRevokeHandler handles POST /settings/api-keys/{id}/revoke
// Auth required. Owner-only delete. HTMX-aware redirect.
func APIKeyRevokeHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		id := e.Request.PathValue("id")
		if id == "" {
			return e.String(http.StatusBadRequest, "missing id")
		}
		coll, err := app.FindCollectionByNameOrId("api_keys")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "api keys collection not available")
		}
		rec, err := app.FindRecordById(coll.Id, id)
		if err != nil || rec == nil {
			return e.String(http.StatusNotFound, "api key not found")
		}
		if rec.GetString("user") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}
		if err := app.Delete(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to revoke api key")
		}
		dest := "/settings?api_key=revoked"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}
