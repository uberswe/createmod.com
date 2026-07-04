package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/ratelimit"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// hmacAuth describes a successfully validated HMAC request.
type hmacAuth struct {
	Timestamp  int64
	ModVersion string
	McUsername string
	Identifier string
}

// authenticateHMAC checks for X-Mod-Message and X-Mod-Signature headers and
// validates them against the shared secret. Returns the parsed auth info on
// success, or nil if the headers are absent or invalid.
func authenticateHMAC(r *http.Request, secret string) *hmacAuth {
	message := strings.TrimSpace(r.Header.Get("X-Mod-Message"))
	signature := strings.TrimSpace(r.Header.Get("X-Mod-Signature"))
	if message == "" || signature == "" {
		return nil
	}
	if !validateModSignature(message, signature, secret) {
		return nil
	}
	timestamp, modVersion, mcUsername, identifier, err := parseModMessage(message, maxModTimestampAge)
	if err != nil {
		return nil
	}
	return &hmacAuth{
		Timestamp:  timestamp,
		ModVersion: modVersion,
		McUsername: mcUsername,
		Identifier: identifier,
	}
}

// requireAPIKeyOrHMAC tries API key auth first, then HMAC auth. Returns:
//   - keyID (non-empty for API key auth, empty for HMAC)
//   - isHMAC (true if authenticated via HMAC)
//   - error (non-nil if both auth methods failed; response already written)
func requireAPIKeyOrHMAC(appStore *store.Store, e *server.RequestEvent, modSecret string) (string, bool, error) {
	// Try API key first
	apiKey := getAPIKeyFromRequest(e.Request)
	if apiKey != "" {
		keyID, ok := verifyAPIKeyFromStore(appStore, apiKey)
		if ok {
			return keyID, false, nil
		}
	}

	// Try HMAC
	if auth := authenticateHMAC(e.Request, modSecret); auth != nil {
		return "", true, nil
	}

	// Neither worked
	_ = writeJSON(e, http.StatusUnauthorized, map[string]string{
		"error": "Authentication required. Use X-API-Key header or HMAC signature (X-Mod-Message + X-Mod-Signature)",
	})
	return "", false, fmt.Errorf("missing authentication")
}

// searchRateLimitAllow enforces a per-minute IP rate limit for HMAC-authenticated
// search requests. Returns (allowed, retryAfterSeconds).
func searchRateLimitAllow(rl ratelimit.Limiter, clientIP string, limit int) (bool, int) {
	if clientIP == "" || rl == nil || limit <= 0 {
		return true, 0
	}
	now := time.Now()
	minuteKey := now.Format("20060102T1504")
	k := "hmac:search:" + clientIP + ":" + minuteKey
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

// apiListResponse is the JSON shape for list/search responses.
type apiListResponse struct {
	Items      []models.Schematic `json:"items"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	HasPrev    bool               `json:"hasPrev"`
	HasNext    bool               `json:"hasNext"`
	Total      int                `json:"total"`
	TotalPages int                `json:"totalPages"`
	Term       string             `json:"term,omitempty"`
}

// getAPIKeyFromRequest extracts API key from the X-API-Key header.
func getAPIKeyFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
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

// serveCachedJSON writes a previously cached 200 JSON body for key and returns
// true on a hit. The cache is per-pod and in-memory; callers pair it with
// writeAndCacheJSON. Only non-personalized responses should be cached this way.
func serveCachedJSON(e *server.RequestEvent, cacheService *cache.Service, key string) bool {
	if cacheService == nil {
		return false
	}
	v, ok := cacheService.Get(key)
	if !ok {
		return false
	}
	body, ok := v.([]byte)
	if !ok {
		return false
	}
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	e.Response.Header().Set("X-Cache", "HIT")
	e.Response.WriteHeader(http.StatusOK)
	_, _ = e.Response.Write(body)
	return true
}

// writeAndCacheJSON marshals data once, caches the bytes under key for ttl
// (per-pod, in-memory), and writes them as a 200 response.
func writeAndCacheJSON(e *server.RequestEvent, cacheService *cache.Service, key string, ttl time.Duration, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if cacheService != nil {
		cacheService.SetWithTTL(key, body, ttl)
	}
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	e.Response.Header().Set("X-Cache", "MISS")
	e.Response.WriteHeader(http.StatusOK)
	_, err = e.Response.Write(body)
	return err
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
// Accepts either X-API-Key or HMAC authentication (X-Mod-Message + X-Mod-Signature headers).
func APISchematicsListHandler(searchEngine search.SearchEngine, rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, modSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, modSecret)
		if err != nil {
			return nil
		}
		if isHMAC {
			// Rate limit HMAC requests by IP: 100/min
			if ok, retry := searchRateLimitAllow(rl, e.RealIP(), 100); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
		} else {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
			if ok, retry := rateLimitAllow(rl, keyID, 120); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
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
		pageSize := parseAPIPerPage(e.Request.URL.Query().Get("per_page"))

		ctx := context.Background()
		var items []models.Schematic
		total := 0
		hasNext := false

		sq := parseAPISearchQuery(e, appStore, cacheService)
		ordered, err := apiSearchResults(ctx, searchEngine, appStore, cacheService, sq, 0)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to fetch schematics")
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

		// Strip internal file paths from public responses.
		for i := range items {
			items[i].SchematicFile = ""
		}

		totalPages := 0
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
		}
		resp := apiListResponse{
			Items:      items,
			Page:       page,
			PageSize:   pageSize,
			HasPrev:    page > 1,
			HasNext:    hasNext,
			Total:      total,
			TotalPages: totalPages,
			Term:       q,
		}

		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		e.Response.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(e.Response).Encode(resp); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	}
}

// parseAPIPerPage clamps the per_page param to a small set of allowed sizes,
// defaulting to 24.
func parseAPIPerPage(raw string) int {
	if raw == "" {
		return 24
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 24
	}
	switch n {
	case 8, 16, 24, 32, 64, 100:
		return n
	}
	return 24
}

// parseAPISearchQuery builds a search.SearchQuery from the public list/search
// query params, mirroring the website search handler (sort, category, mcv, cv,
// rating, tag, mods). When no term and no sort are given it browses by trending.
func parseAPISearchQuery(e *server.RequestEvent, appStore *store.Store, cacheService *cache.Service) search.SearchQuery {
	get := e.Request.URL.Query().Get
	q := get("query")
	if q == "" {
		q = get("q")
	}
	term := strings.ReplaceAll(strings.TrimSpace(q), "-", " ")

	// Only treat sort as set when it parses to a valid order (1..8). An invalid
	// or non-numeric sort is ignored so it can't override the trending default
	// for empty-term browsing or silently produce arbitrary relevancy order.
	order := search.BestMatchOrder
	hasSort := false
	if n, err := strconv.Atoi(get("sort")); err == nil && n >= search.BestMatchOrder && n <= search.TrendingOrder {
		order = n
		hasSort = true
	}
	if term == "" && !hasSort {
		order = search.TrendingOrder
	}

	// Rating filter only applies for a valid minimum in 0..5; anything else
	// (e.g. rating=10) is ignored rather than silently matching nothing.
	rating := -1
	if v := get("rating"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 5 {
			rating = n
		}
	}
	category := "all"
	if v := get("category"); v != "" {
		category = v
	}
	mcVersion := "all"
	if v := get("mcv"); v != "" {
		mcVersion = v
	}
	createVersion := "all"
	if v := get("cv"); v != "" {
		createVersion = v
	}

	var selectedTags []string
	if tp := get("tag"); tp != "" && tp != "all" {
		for _, t := range strings.Split(tp, ",") {
			if t = strings.TrimSpace(t); t != "" {
				selectedTags = append(selectedTags, t)
			}
		}
	}

	// Accept both comma-separated "mods" and repeated "mod" params, then resolve
	// the mod namespaces to the display names the search index stores.
	var selectedMods []string
	if mp := get("mods"); mp != "" {
		for _, m := range strings.Split(mp, ",") {
			if m = strings.TrimSpace(m); m != "" {
				selectedMods = append(selectedMods, m)
			}
		}
	}
	if len(selectedMods) == 0 {
		for _, m := range e.Request.URL.Query()["mod"] {
			if m = strings.TrimSpace(m); m != "" {
				selectedMods = append(selectedMods, m)
			}
		}
	}
	var meiliModNames []string
	if len(selectedMods) > 0 {
		allMods := allModOptionsFromStore(appStore, cacheService)
		nsToDisplay := make(map[string]string, len(allMods))
		for _, mo := range allMods {
			nsToDisplay[mo.Namespace] = mo.DisplayName
		}
		for _, ns := range selectedMods {
			if dn, ok := nsToDisplay[ns]; ok {
				meiliModNames = append(meiliModNames, dn)
			}
		}
	}

	// Expand a "~6.0" major-version group into its individual versions.
	var createVersionList []string
	if strings.HasPrefix(createVersion, "~") {
		prefix := strings.TrimPrefix(createVersion, "~")
		for _, cv := range allCreatemodVersionsFromStore(appStore) {
			if createVersionMajor(cv.Version) == prefix {
				createVersionList = append(createVersionList, cv.Version)
			}
		}
	}

	return search.SearchQuery{
		Term:             term,
		Order:            order,
		Rating:           rating,
		Category:         category,
		Tags:             selectedTags,
		MinecraftVersion: mcVersion,
		CreateVersion:    createVersion,
		CreateVersions:   createVersionList,
		Mods:             meiliModNames,
	}
}

// searchFetchCushion is how many extra IDs beyond a caller's limit apiSearchResults
// hydrates, so deleted/non-public rows filtered out afterwards rarely leave the
// caller short of its requested count.
const searchFetchCushion = 24

// apiSearchResults runs a search and returns the matching public schematics in
// search-result order (deleted and non-public states filtered out).
//
// limit caps how many results are hydrated: when > 0, only the top IDs (plus a
// small cushion to absorb filtered-out rows) are fetched from the store, so
// callers that need just a rail of N items (e.g. the home page) don't pull the
// full search result set — up to 5000 rows — from Postgres. Pass 0 to hydrate
// everything (the list handler needs the full set to compute an accurate total).
func apiSearchResults(ctx context.Context, searchEngine search.SearchEngine, appStore *store.Store, cacheService *cache.Service, sq search.SearchQuery, limit int) ([]store.Schematic, error) {
	ids, _ := searchEngine.Search(ctx, sq)
	if len(ids) == 0 {
		return nil, nil
	}
	// Fetch a cushion beyond limit so deleted/non-public filtering below still
	// leaves at least `limit` items in the common case.
	if limit > 0 && len(ids) > limit+searchFetchCushion {
		ids = ids[:limit+searchFetchCushion]
	}
	storeSchematics, err := appStore.Schematics.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]store.Schematic, len(storeSchematics))
	for _, s := range storeSchematics {
		byID[s.ID] = s
	}
	ordered := make([]store.Schematic, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			if s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
				continue
			}
			ordered = append(ordered, s)
		}
	}
	return ordered, nil
}

// APISchematicDetailHandler serves GET /api/schematics/{name} returning one schematic by name.
// Accepts either X-API-Key or HMAC authentication (X-Mod-Message + X-Mod-Signature headers).
func APISchematicDetailHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, modSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, modSecret)
		if err != nil {
			return nil
		}
		if isHMAC {
			// Rate limit HMAC requests by IP: 100/min
			if ok, retry := searchRateLimitAllow(rl, e.RealIP(), 100); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
		} else {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
			if ok, retry := rateLimitAllow(rl, keyID, 120); !ok {
				e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
		}
		name := e.Request.PathValue("name")
		if strings.TrimSpace(name) == "" {
			return e.String(http.StatusBadRequest, "missing schematic name")
		}
		// Detail is the same for every caller; serve a short-lived cached body.
		cacheKey := "api:schematic:" + name
		if serveCachedJSON(e, cacheService, cacheKey) {
			return nil
		}
		ctx := context.Background()
		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		if s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			e.Response.WriteHeader(http.StatusNotFound)
			_, _ = e.Response.Write([]byte(`{"error":"not found"}`))
			return nil
		}
		item := MapStoreSchematicToModel(appStore, *s, cacheService)
		item.SchematicFile = ""
		return writeAndCacheJSON(e, cacheService, cacheKey, 120*time.Second, item)
	}
}
