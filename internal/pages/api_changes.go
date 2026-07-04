package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"encoding/base64"
	"net/http"
	"strings"
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

// encodeChangesCursor packs the compound keyset position (timestamp, kind, name)
// into one opaque token. name is last so a name containing the separator can't
// corrupt the other fields (kind is a fixed vocabulary, the timestamp has no '|').
func encodeChangesCursor(at time.Time, name, kind string) string {
	s := at.UTC().Format(time.RFC3339Nano) + "|" + kind + "|" + name
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// decodeChangesCursor parses a cursor token back into (at, name, kind). It also
// accepts a bare RFC3339 timestamp for backwards compatibility with the original
// cursor format (treated as position at the start of that timestamp).
func decodeChangesCursor(param string) (at time.Time, name, kind string, ok bool) {
	if t, err := time.Parse(time.RFC3339Nano, param); err == nil {
		return t.UTC(), "", "", true
	}
	raw, err := base64.RawURLEncoding.DecodeString(param)
	if err != nil {
		return time.Time{}, "", "", false
	}
	parts := strings.SplitN(string(raw), "|", 3)
	if len(parts) != 3 {
		return time.Time{}, "", "", false
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", "", false
	}
	return t.UTC(), parts[2], parts[1], true
}

// APISchematicChangesHandler serves GET /api/schematics/changes?cursor=<token>.
// It returns schematics edited or removed after the cursor so external caches
// can invalidate precisely. Edits are read from the schematic_versions history
// and removals from the deleted timestamp; the cursor is an opaque token
// encoding a (timestamp, name, kind) keyset position. Call without a cursor to
// get the current cursor (with an empty list) and start from now. Public (no
// API key) since it exposes only names.
func APISchematicChangesHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		ctx := context.Background()
		now := time.Now().UTC()

		cursorParam := e.Request.URL.Query().Get("cursor")
		if cursorParam == "" {
			return writeJSON(e, http.StatusOK, apiChangesResponse{
				Changes: []apiChangeItem{},
				Cursor:  encodeChangesCursor(now, "", ""),
			})
		}
		sinceAt, sinceName, sinceKind, ok := decodeChangesCursor(cursorParam)
		if !ok {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}

		const limit = 500
		changes, err := appStore.Schematics.ChangesSince(ctx, sinceAt, sinceName, sinceKind, limit+1)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to list changes"})
		}
		hasMore := len(changes) > limit
		if hasMore {
			changes = changes[:limit]
		}

		items := make([]apiChangeItem, 0, len(changes))
		// With no changes, advance the cursor to now so the next poll starts
		// fresh; otherwise the cursor is the last row's compound keyset position.
		cursorAt, cursorName, cursorKind := now, "", ""
		for _, c := range changes {
			items = append(items, apiChangeItem{Name: c.Name, Kind: c.Kind})
			cursorAt, cursorName, cursorKind = c.At, c.Name, c.Kind
		}
		return writeJSON(e, http.StatusOK, apiChangesResponse{
			Changes: items,
			Cursor:  encodeChangesCursor(cursorAt, cursorName, cursorKind),
			HasMore: hasMore,
		})
	}
}
