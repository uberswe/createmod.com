package pages

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SchematicAddToCollectionHandler handles POST /schematics/{name}/add-to-collection
// Minimal, best-effort implementation:
//   - Requires auth
//   - Resolves schematic by {name}
//   - Resolves collection by provided slug or id
//   - Tries to create a link record in one of the expected join collections
//     ("collections_schematics" or "collection_schematics") with fields
//     {collection, schematic}. If not present, attempts to append schematic id
//     to a multi-rel field "schematics" on the target collection.
//   - HTMX-aware redirect back to the schematic page.
func SchematicAddToCollectionHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		name := e.Request.PathValue("name")
		returnTo := "/schematics/" + name
		if e.Auth == nil {
			// Require login
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		collInput := e.Request.FormValue("collection")
		if collInput == "" {
			// Nothing to do
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", returnTo+"?error=missing_collection")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, returnTo+"?error=missing_collection")
		}

		// Resolve schematic record by name
		schemColl, err := app.FindCollectionByNameOrId("schematics")
		if err != nil || schemColl == nil {
			return e.String(http.StatusInternalServerError, "schematics collection not available")
		}
		schemRecs, err := app.FindRecordsByFilter(schemColl.Id, "name = {:name}", "-created", 1, 0, dbx.Params{"name": name})
		if err != nil || len(schemRecs) == 0 {
			return e.String(http.StatusNotFound, "schematic not found")
		}
		schematic := schemRecs[0]

		// Resolve collection by slug first, then id
		collsColl, err := app.FindCollectionByNameOrId("collections")
		if err != nil || collsColl == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		var collection *core.Record
		if recs, err := app.FindRecordsByFilter(collsColl.Id, "slug = {:slug}", "-created", 1, 0, dbx.Params{"slug": collInput}); err == nil && len(recs) > 0 {
			collection = recs[0]
		}
		if collection == nil {
			if rec, err := app.FindRecordById(collsColl.Id, collInput); err == nil {
				collection = rec
			}
		}
		if collection == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", returnTo+"?error=collection_not_found")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, returnTo+"?error=collection_not_found")
		}

		// Try join table variants
		joinNames := []string{"collections_schematics", "collection_schematics"}
		linked := false
		for _, jn := range joinNames {
			if joinColl, jerr := app.FindCollectionByNameOrId(jn); jerr == nil && joinColl != nil {
				// avoid duplicate
				if existing, _ := app.FindRecordsByFilter(joinColl.Id, "collection = {:c} && schematic = {:s}", "-created", 1, 0, dbx.Params{"c": collection.Id, "s": schematic.Id}); len(existing) > 0 {
					linked = true
					break
				}
				rec := core.NewRecord(joinColl)
				rec.Set("collection", collection.Id)
				rec.Set("schematic", schematic.Id)
				if saveErr := app.Save(rec); saveErr == nil {
					linked = true
					break
				} else {
					// continue to try other strategies
					app.Logger().Warn("add-to-collection: join save failed", "error", saveErr, "join", jn)
				}
			}
		}

		if !linked {
			// Try appending to a multi-rel field on the collection: schematics
			ids := collection.GetStringSlice("schematics")
			// Prevent duplicates
			already := false
			for _, id := range ids {
				if id == schematic.Id {
					already = true
					break
				}
			}
			if !already {
				ids = append(ids, schematic.Id)
			}
			collection.Set("schematics", ids)
			if err := app.Save(collection); err == nil {
				linked = true
			} else {
				app.Logger().Warn("add-to-collection: fallback save failed", "error", err)
			}
		}

		// Redirect back with status flag
		suffix := "?added=1"
		if !linked {
			suffix = "?error=unsupported"
		}
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", returnTo+suffix)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, returnTo+suffix)
	}
}
