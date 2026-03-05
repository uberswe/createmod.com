package pages

import (
	"context"
	"createmod/internal/store"
	"net/http"
	"strings"

	"createmod/internal/server"
)

// CollectionsReorderHandler handles POST /collections/{slug}/reorder
// Accepts a comma-separated list of schematic IDs via form field "schematics" (or alias "ids").
// Author-only. Clears existing join table entries and re-creates them with position ordering.
func CollectionsReorderHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		slug := e.Request.PathValue("slug")
		if slug == "" {
			return e.String(http.StatusBadRequest, "missing slug")
		}
		if ok, err := requireAuth(e); !ok {
			return err
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		list := strings.TrimSpace(e.Request.FormValue("schematics"))
		if list == "" {
			list = strings.TrimSpace(e.Request.FormValue("ids"))
		}
		if list == "" {
			return e.String(http.StatusBadRequest, "missing schematics list")
		}
		// Parse comma-separated ids; trim blanks and dedupe while preserving order
		raw := strings.Split(list, ",")
		ordered := make([]string, 0, len(raw))
		seen := make(map[string]struct{}, len(raw))
		for _, r := range raw {
			id := strings.TrimSpace(r)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ordered = append(ordered, id)
		}
		if len(ordered) == 0 {
			return e.String(http.StatusBadRequest, "no valid ids provided")
		}

		ctx := context.Background()

		// Resolve collection by slug first, then by id
		coll, err := appStore.Collections.GetBySlug(ctx, slug)
		if err != nil || coll == nil {
			coll, err = appStore.Collections.GetByID(ctx, slug)
		}
		if coll == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		// Author-only
		if coll.AuthorID == nil || *coll.AuthorID != authenticatedUserID(e) {
			return e.String(http.StatusForbidden, "not allowed")
		}

		// Clear existing associations and re-add with position
		_ = appStore.Collections.ClearSchematics(ctx, coll.ID)
		for i, id := range ordered {
			_ = appStore.Collections.AddSchematic(ctx, coll.ID, id, i+1)
		}

		dest := "/collections/" + slug + "/edit"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
