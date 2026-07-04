package pages

import (
	"context"
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strings"
	"time"
)

// apiCommentsResponse is the JSON shape for GET /api/schematics/{name}/comments.
type apiCommentsResponse struct {
	Count    int              `json:"count"`
	Comments []models.Comment `json:"comments"`
}

// APISchematicCommentsHandler serves GET /api/schematics/{name}/comments,
// returning the approved comment thread for a schematic. Auth: API key or HMAC.
func APISchematicCommentsHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		const endpoint = "GET /api/schematics/{name}/comments"
		keyID, isHMAC, err := requireAPIKeyOrHMAC(appStore, e, cacheService)
		if err != nil {
			return nil
		}
		if rejected := applyAPIRateLimit(e, rl, keyID, isHMAC); rejected {
			return nil
		}
		if !isHMAC {
			defer func() { recordAPIKeyUsageStore(appStore, keyID, endpoint) }()
		}

		name := e.Request.PathValue("name")
		if strings.TrimSpace(name) == "" {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing schematic name"})
		}

		// Comment threads are the same for every caller; cache briefly.
		cacheKey := "api:comments:" + name
		if serveCachedJSON(e, cacheService, cacheKey) {
			return nil
		}

		ctx := context.Background()
		s, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || s == nil || s.Deleted != nil || !store.IsPublicState(s.ModerationState) {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": "not found"})
		}

		comments := findSchematicCommentsFromStore(appStore, s.ID, nil, cacheService, "")
		if comments == nil {
			comments = []models.Comment{}
		}
		return writeAndCacheJSON(e, cacheService, cacheKey, 60*time.Second, apiCommentsResponse{Count: len(comments), Comments: comments})
	}
}
