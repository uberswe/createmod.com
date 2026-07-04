package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"time"
)

type apiChangeItem struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type apiChangesResponse struct {
	Changes []apiChangeItem `json:"changes"`
	Cursor  string          `json:"cursor"`
	HasMore bool            `json:"hasMore"`
}

// APISchematicChangesHandler serves GET /api/schematics/changes?cursor=<ts>.
// It returns schematics edited or removed after the cursor so external caches
// can invalidate precisely. Edits are read from the schematic_versions history
// and removals from the deleted timestamp; the cursor is an opaque RFC3339
// timestamp. Call without a cursor to get the current cursor (with an empty
// list) and start from now. Public (no API key) since it exposes only names.
func APISchematicChangesHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := context.Background()
		now := time.Now().UTC()

		cursorParam := e.Request.URL.Query().Get("cursor")
		if cursorParam == "" {
			return writeJSON(e, http.StatusOK, apiChangesResponse{
				Changes: []apiChangeItem{},
				Cursor:  now.Format(time.RFC3339Nano),
			})
		}
		since, err := time.Parse(time.RFC3339Nano, cursorParam)
		if err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}

		const limit = 500
		changes, err := appStore.Schematics.ChangesSince(ctx, since, limit+1)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to list changes"})
		}
		hasMore := len(changes) > limit
		if hasMore {
			changes = changes[:limit]
		}

		items := make([]apiChangeItem, 0, len(changes))
		cursor := now
		for _, c := range changes {
			items = append(items, apiChangeItem{Name: c.Name, Kind: c.Kind})
			cursor = c.At
		}
		return writeJSON(e, http.StatusOK, apiChangesResponse{
			Changes: items,
			Cursor:  cursor.UTC().Format(time.RFC3339Nano),
			HasMore: hasMore,
		})
	}
}
