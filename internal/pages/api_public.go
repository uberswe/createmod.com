package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// apiListResponse is the JSON shape for list/search responses.
type apiListResponse struct {
	Items    []models.Schematic `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	HasPrev  bool               `json:"hasPrev"`
	HasNext  bool               `json:"hasNext"`
	Total    int                `json:"total"`
	Term     string             `json:"term,omitempty"`
}

// getAPIKeyFromRequest extracts API key from header or query param.
func getAPIKeyFromRequest(r *http.Request) string {
	key := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if key == "" {
		key = strings.TrimSpace(r.URL.Query().Get("api_key"))
	}
	return key
}

// verifyAPIKey looks up the API key by hashing the plaintext and matching api_keys.key_hash.
// Returns the api_keys record id if valid.
func verifyAPIKey(app *pocketbase.PocketBase, plaintext string) (string, bool) {
	if strings.TrimSpace(plaintext) == "" {
		return "", false
	}
	sum := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(sum[:])
	coll, err := app.FindCollectionByNameOrId("api_keys")
	if err != nil || coll == nil {
		return "", false
	}
	recs, err := app.FindRecordsByFilter(coll.Id, "key_hash = {:h}", "-created", 1, 0, dbx.Params{"h": hash})
	if err != nil || len(recs) == 0 {
		return "", false
	}
	return recs[0].Id, true
}

// writeJSON is a small helper for JSON error/success responses.
func writeJSON(e *core.RequestEvent, status int, data interface{}) error {
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	e.Response.WriteHeader(status)
	return json.NewEncoder(e.Response).Encode(data)
}

// requireAPIKey extracts and validates the API key from the request.
// Returns (keyID, nil) on success or ("", error-already-written) on failure.
func requireAPIKey(app *pocketbase.PocketBase, e *core.RequestEvent) (string, error) {
	apiKey := getAPIKeyFromRequest(e.Request)
	if apiKey == "" {
		_ = writeJSON(e, http.StatusUnauthorized, map[string]string{
			"error": "API key required. Get one at /settings/api-keys",
		})
		return "", fmt.Errorf("missing api key")
	}
	keyID, ok := verifyAPIKey(app, apiKey)
	if !ok {
		_ = writeJSON(e, http.StatusUnauthorized, map[string]string{
			"error": "invalid API key",
		})
		return "", fmt.Errorf("invalid api key")
	}
	return keyID, nil
}

// recordAPIKeyUsage increments counters in api_key_usage for the provided key id and endpoint.
func recordAPIKeyUsage(app *pocketbase.PocketBase, keyID string, endpoint string, isError bool) {
	coll, err := app.FindCollectionByNameOrId("api_key_usage")
	if err != nil || coll == nil {
		return
	}
	recs, _ := app.FindRecordsByFilter(coll.Id, "key = {:k} && endpoint = {:ep}", "-created", 1, 0, dbx.Params{"k": keyID, "ep": endpoint})
	if len(recs) == 0 {
		r := core.NewRecord(coll)
		r.Set("key", keyID)
		r.Set("endpoint", endpoint)
		r.Set("total_requests", 1)
		r.Set("total_errors", 0)
		if isError {
			r.Set("total_errors", 1)
		}
		_ = app.Save(r)
		return
	}
	r := recs[0]
	r.Set("total_requests", r.GetInt("total_requests")+1)
	if isError {
		r.Set("total_errors", r.GetInt("total_errors")+1)
	}
	_ = app.Save(r)
}

// rateLimitAllow enforces a simple per-minute limit per API key id using the in-memory cache.
// Returns (allowed, retryAfterSeconds).
func rateLimitAllow(cacheService *cache.Service, keyID string, limit int) (bool, int) {
	if keyID == "" || limit <= 0 {
		return true, 0
	}
	now := time.Now()
	// key is rounded to minute for a sliding window approximation
	minuteKey := now.Format("20060102T1504")
	k := "rl:" + keyID + ":" + minuteKey
	cur, _ := cacheService.GetInt(k)
	cur++
	// TTL until the end of current minute
	ttl := time.Until(now.Truncate(time.Minute).Add(time.Minute))
	if ttl <= 0 {
		ttl = time.Second
	}
	cacheService.SetWithTTL(k, cur, ttl)
	if cur > limit {
		ra := int(ttl.Seconds())
		if ra < 1 {
			ra = 1
		}
		return false, ra
	}
	return true, 0
}

// APISchematicsListHandler serves GET /api/schematics as a simple JSON API for searching/listing schematics.
// Query parameters:
//   - query (or q): search term; if absent, returns newest schematics
//   - page (or p): 1-based page index (default 1)
func APISchematicsListHandler(app *pocketbase.PocketBase, searchService *search.Service, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		const endpoint = "GET /api/schematics"
		keyID, err := requireAPIKey(app, e)
		if err != nil {
			return nil
		}
		success := true
		defer func() { recordAPIKeyUsage(app, keyID, endpoint, !success) }()
		if ok, retry := rateLimitAllow(cacheService, keyID, 120); !ok {
			success = false
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}
		q := e.Request.URL.Query().Get("query")
		if q == "" {
			q = e.Request.URL.Query().Get("q")
		}
		page := 1
		if v := e.Request.URL.Query().Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		if v := e.Request.URL.Query().Get("p"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		pageSize := 24

		var items []models.Schematic
		total := 0
		hasNext := false

		if strings.TrimSpace(q) == "" {
			// Fallback: newest schematics
			coll, err := app.FindCollectionByNameOrId("schematics")
			if err != nil || coll == nil {
				success = false
				return e.String(http.StatusInternalServerError, "schematics collection not available")
			}
			limit := pageSize + 1
			offset := (page - 1) * pageSize
			recs, err := app.FindRecordsByFilter(coll.Id, "deleted = '' && moderated = true && (blacklisted = null || blacklisted = false) && (scheduled_at = null || scheduled_at <= {:now})", "-created", limit, offset, dbx.Params{"now": time.Now()})
			if err != nil {
				success = false
				return e.String(http.StatusInternalServerError, "failed to list schematics")
			}
			hasNext = len(recs) > pageSize
			if hasNext {
				recs = recs[:pageSize]
			}
			items = MapResultsToSchematic(app, recs, cacheService)
			// We don't know the exact total cheaply here; return a best-effort of known window
			total = (page-1)*pageSize + len(items)
		} else {
			// Search via in-memory searchService, then DB fetch in order
			term := strings.ReplaceAll(q, "-", " ")
			ids := searchService.Search(term, search.MostViewedOrder, -1, "all", nil, "all", "all")
			// Fetch matching records
			var res []*core.Record
			if len(ids) > 0 {
				iface := make([]interface{}, 0, len(ids))
				for _, id := range ids {
					iface = append(iface, id)
				}
				err := app.RecordQuery("schematics").
					Select("schematics.*").
					From("schematics").
					Where(dbx.In("id", iface...)).
					All(&res)
				if err != nil {
					success = false
					return e.String(http.StatusInternalServerError, "failed to fetch schematics")
				}
			}
			// Preserve search order
			ordered := make([]*core.Record, 0, len(res))
			for i := range ids {
				for j := range res {
					if res[j].Id == ids[i] {
						ordered = append(ordered, res[j])
					}
				}
			}
			total = len(ordered)
			// Pagination slice
			start := (page - 1) * pageSize
			if start < 0 {
				start = 0
			}
			end := start + pageSize
			if end > total {
				end = total
			}
			cur := []*core.Record{}
			if total > 0 && start < total {
				cur = ordered[start:end]
			}
			hasNext = end < total
			items = MapResultsToSchematic(app, cur, cacheService)
		}

		resp := apiListResponse{
			Items:    items,
			Page:     page,
			PageSize: pageSize,
			HasPrev:  page > 1,
			HasNext:  hasNext,
			Total:    total,
			Term:     q,
		}

		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(e.Response).Encode(resp); err != nil {
			success = false
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	}
}

// APISchematicDetailHandler serves GET /api/schematics/{name} returning one schematic by name.
func APISchematicDetailHandler(app *pocketbase.PocketBase, cacheService *cache.Service, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}"
		keyID, err := requireAPIKey(app, e)
		if err != nil {
			return nil
		}
		success := true
		defer func() { recordAPIKeyUsage(app, keyID, endpoint, !success) }()
		if ok, retry := rateLimitAllow(cacheService, keyID, 120); !ok {
			success = false
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}
		name := e.Request.PathValue("name")
		if strings.TrimSpace(name) == "" {
			success = false
			return e.String(http.StatusBadRequest, "missing schematic name")
		}
		coll, err := app.FindCollectionByNameOrId("schematics")
		if err != nil || coll == nil {
			success = false
			return e.String(http.StatusInternalServerError, "schematics collection not available")
		}
		recs, err := app.FindRecordsByFilter(coll.Id, "name = {:name} && deleted = '' && moderated = true && (blacklisted = null || blacklisted = false) && (scheduled_at = null || scheduled_at <= {:now})", "-created", 1, 0, dbx.Params{"name": name, "now": time.Now()})
		if err != nil {
			success = false
			return e.String(http.StatusInternalServerError, "failed to query schematic")
		}
		if len(recs) == 0 {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		items := MapResultsToSchematic(app, recs, cacheService)
		if len(items) == 0 {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(e.Response).Encode(items[0]); err != nil {
			success = false
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	}
}
