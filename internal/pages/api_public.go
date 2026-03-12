package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/ratelimit"
	"createmod/internal/search"
	"createmod/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"createmod/internal/server"
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

// verifyAPIKeyFromStore looks up the API key by its last 8 characters,
// then verifies by comparing the full sha256 hash.
func verifyAPIKeyFromStore(appStore *store.Store, plaintext string) (string, bool) {
	if strings.TrimSpace(plaintext) == "" {
		return "", false
	}
	last8 := plaintext
	if len(plaintext) >= 8 {
		last8 = plaintext[len(plaintext)-8:]
	}
	ctx := context.Background()
	key, err := appStore.APIKeys.GetByLast8(ctx, last8)
	if err != nil || key == nil {
		return "", false
	}
	// Verify the full hash
	sum := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(sum[:])
	if key.KeyHash != hash {
		return "", false
	}
	return key.ID, true
}

// writeJSON is a small helper for JSON error/success responses.
func writeJSON(e *server.RequestEvent, status int, data interface{}) error {
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	e.Response.WriteHeader(status)
	return json.NewEncoder(e.Response).Encode(data)
}

// requireAPIKeyFromStore extracts and validates the API key from the request using store.
func requireAPIKeyFromStore(appStore *store.Store, e *server.RequestEvent) (string, error) {
	apiKey := getAPIKeyFromRequest(e.Request)
	if apiKey == "" {
		_ = writeJSON(e, http.StatusUnauthorized, map[string]string{
			"error": "API key required. Get one at /settings/api-keys",
		})
		return "", fmt.Errorf("missing api key")
	}
	keyID, ok := verifyAPIKeyFromStore(appStore, apiKey)
	if !ok {
		_ = writeJSON(e, http.StatusUnauthorized, map[string]string{
			"error": "invalid API key",
		})
		return "", fmt.Errorf("invalid api key")
	}
	return keyID, nil
}

// recordAPIKeyUsageStore increments counters for the provided key id and endpoint.
func recordAPIKeyUsageStore(appStore *store.Store, keyID string, endpoint string) {
	ctx := context.Background()
	_ = appStore.APIKeys.LogUsage(ctx, keyID, endpoint)
}

// rateLimitAllow enforces a simple per-minute limit per API key id using the rate limiter.
// Returns (allowed, retryAfterSeconds).
func rateLimitAllow(rl ratelimit.Limiter, keyID string, limit int) (bool, int) {
	if keyID == "" || limit <= 0 || rl == nil {
		return true, 0
	}
	now := time.Now()
	minuteKey := now.Format("20060102T1504")
	k := "rl:" + keyID + ":" + minuteKey
	ttl := time.Until(now.Truncate(time.Minute).Add(time.Minute))
	if ttl <= 0 {
		ttl = time.Second
	}
	ok, _ := rl.Allow(context.Background(), k, limit, ttl)
	if !ok {
		ra := int(ttl.Seconds())
		if ra < 1 {
			ra = 1
		}
		return false, ra
	}
	return true, 0
}

// APISchematicsListHandler serves GET /api/schematics as a simple JSON API for searching/listing schematics.
func APISchematicsListHandler(searchService *search.Service, rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics"
		keyID, err := requireAPIKeyFromStore(appStore, e)
		if err != nil {
			return nil
		}
		defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
		if ok, retry := rateLimitAllow(rl, keyID, 120); !ok {
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

		ctx := context.Background()
		var items []models.Schematic
		total := 0
		hasNext := false

		if strings.TrimSpace(q) == "" {
			// Fallback: newest schematics via store
			limit := pageSize + 1
			offset := (page - 1) * pageSize
			schematics, err := appStore.Schematics.ListApproved(ctx, limit, offset)
			if err != nil {
				return e.String(http.StatusInternalServerError, "failed to list schematics")
			}
			hasNext = len(schematics) > pageSize
			if hasNext {
				schematics = schematics[:pageSize]
			}
			items = MapStoreSchematics(appStore, schematics, cacheService)
			total = (page-1)*pageSize + len(items)
		} else {
			// Search via in-memory searchService, then store fetch in order
			term := strings.ReplaceAll(q, "-", " ")
			ids := searchService.Search(term, search.MostViewedOrder, -1, "all", nil, "all", "all", false)
			if len(ids) > 0 {
				storeSchematics, err := appStore.Schematics.ListByIDs(ctx, ids)
				if err != nil {
					return e.String(http.StatusInternalServerError, "failed to fetch schematics")
				}
				// Preserve search order
				byID := make(map[string]store.Schematic, len(storeSchematics))
				for _, s := range storeSchematics {
					byID[s.ID] = s
				}
				ordered := make([]store.Schematic, 0, len(ids))
				for _, id := range ids {
					if s, ok := byID[id]; ok {
						ordered = append(ordered, s)
					}
				}
				total = len(ordered)
				start := (page - 1) * pageSize
				if start < 0 {
					start = 0
				}
				end := start + pageSize
				if end > total {
					end = total
				}
				var cur []store.Schematic
				if total > 0 && start < total {
					cur = ordered[start:end]
				}
				hasNext = end < total
				items = MapStoreSchematics(appStore, cur, cacheService)
			}
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
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	}
}

// APISchematicDetailHandler serves GET /api/schematics/{name} returning one schematic by name.
func APISchematicDetailHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}"
		keyID, err := requireAPIKeyFromStore(appStore, e)
		if err != nil {
			return nil
		}
		defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
		if ok, retry := rateLimitAllow(rl, keyID, 120); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}
		name := e.Request.PathValue("name")
		if strings.TrimSpace(name) == "" {
			return e.String(http.StatusBadRequest, "missing schematic name")
		}
		ctx := context.Background()
		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		if s.Deleted != nil || !s.Moderated || s.Blacklisted {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		item := MapStoreSchematicToModel(appStore, *s, cacheService)
		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(e.Response).Encode(item); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	}
}
