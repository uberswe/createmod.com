package pages

import (
	"createmod/internal/store"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// CollectionsReorderHandler handles POST /collections/{slug}/reorder
// Minimal implementation to support drag-and-drop reorder in the future.
// Accepts a comma-separated list of schematic IDs via form field "schematics" (or alias "ids").
// Author-only. Best-effort persists order to a multi-rel field on the collection
// and, if a join table exists with a numeric "position" field, updates those as well.
func CollectionsReorderHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
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

		// Resolve collection by slug first, then by id
		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		var rec *core.Record
		if r, err := app.FindRecordsByFilter(coll.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": slug}); err == nil && len(r) > 0 {
			rec = r[0]
		}
		if rec == nil {
			if r, err := app.FindRecordById(coll.Id, slug); err == nil {
				rec = r
			}
		}
		if rec == nil {
			return e.String(http.StatusNotFound, "collection not found")
		}
		// Author-only
		if rec.GetString("author") != authenticatedUserID(e) {
			return e.String(http.StatusForbidden, "not allowed")
		}

		// Best-effort: set the multi-rel field "schematics" to the ordered list
		// (PocketBase will enforce relation validity; unknown ids will error out, so we attempt save and fallback.)
		{
			original := rec.GetStringSlice("schematics")
			rec.Set("schematics", ordered)
			if err := app.Save(rec); err != nil {
				// restore on failure (field may not exist in schema)
				rec.Set("schematics", original)
			}
		}

		// Best-effort: update join table positions if available
		for _, jn := range []string{"collections_schematics", "collection_schematics"} {
			if jcoll, jerr := app.FindCollectionByNameOrId(jn); jerr == nil && jcoll != nil {
				// Load existing links for this collection
				links, _ := app.FindRecordsByFilter(jcoll.Id, "collection = {:c}", "-created", 5000, 0, dbx.Params{"c": rec.Id})
				if len(links) == 0 {
					continue
				}
				// Map for quick lookup
				byS := make(map[string]*core.Record, len(links))
				for _, l := range links {
					byS[l.GetString("schematic")] = l
				}
				posField := "position"
				// Try to set position if the field is present; PocketBase Save will fail otherwise.
				for i, id := range ordered {
					if link, ok := byS[id]; ok {
						// 1-based position for human-friendly ordering
						link.Set(posField, i+1)
						_ = app.Save(link) // ignore errors; schema may not have the field
					}
				}
			}
		}

		dest := "/collections/" + slug + "/edit"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
	}
}
