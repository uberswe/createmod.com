package pages

import (
	"createmod/internal/cache"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtempl "github.com/pocketbase/pocketbase/tools/template"
	"net/http"
	"sort"
	"time"
)

var collectionsEditTemplates = append([]string{
	"./template/collections_edit.html",
}, commonTemplates...)

type CollectionsEditData struct {
	DefaultData
	TitleText    string
	Description  string
	BannerURL    string
	Error        string
	SchematicIDs []string
}

// CollectionsEditHandler renders the edit form for a collection (author-only).
func CollectionsEditHandler(app *pocketbase.PocketBase, registry *pbtempl.Registry, cacheService *cache.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", "/login")
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		slug := e.Request.PathValue("slug")
		d := CollectionsEditData{}
		d.Populate(e)
		d.Categories = allCategories(app, cacheService)
		d.Slug = "/collections/" + slug

		coll, err := app.FindCollectionByNameOrId("collections")
		if err != nil || coll == nil {
			return e.String(http.StatusInternalServerError, "collections collection not available")
		}
		// Find by slug first, fallback to id
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
		if rec.GetString("author") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}

		d.TitleText = rec.GetString("title")
		if d.TitleText == "" {
			d.TitleText = rec.GetString("name")
		}
		d.Description = rec.GetString("description")
		d.BannerURL = rec.GetString("banner_url")
		d.Title = "Edit collection"
		d.Description = "Update your collection"

		// Discover associated schematics to power the reorder UI.
		// Preference:
		//  1) If join table exists with optional numeric position, use it (ascending by position if present).
		//  2) Else use the collection's multi-rel field "schematics" as-is.
		ids := make([]string, 0, 64)
		// Start with multi-rel field as fallback.
		if rel := rec.GetStringSlice("schematics"); len(rel) > 0 {
			// copy to avoid mutating underlying slice
			tmp := make([]string, 0, len(rel))
			seen := make(map[string]struct{}, len(rel))
			for _, s := range rel {
				if s == "" {
					continue
				}
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
				tmp = append(tmp, s)
			}
			ids = tmp
		}
		// Try join associations
		type pair struct {
			sid string
			pos int
			idx int
		}
		best := make([]pair, 0, 128)
		for _, jn := range []string{"collections_schematics", "collection_schematics"} {
			if jcoll, jerr := app.FindCollectionByNameOrId(jn); jerr == nil && jcoll != nil {
				// Load links. Sort by -created to get deterministic latest-first; we'll re-sort by position if present.
				if links, _ := app.FindRecordsByFilter(jcoll.Id, "collection = {:c}", "-created", 5000, 0, dbx.Params{"c": rec.Id}); len(links) > 0 {
					best = best[:0]
					seen := make(map[string]struct{}, len(links))
					for i, l := range links {
						sid := l.GetString("schematic")
						if sid == "" {
							continue
						}
						if _, ok := seen[sid]; ok {
							continue
						}
						seen[sid] = struct{}{}
						p := l.GetInt("position")
						best = append(best, pair{sid: sid, pos: p, idx: i})
					}
					// If any position > 0, sort by pos ascending then idx to stabilize.
					anyPos := false
					for _, it := range best {
						if it.pos > 0 {
							anyPos = true
							break
						}
					}
					if anyPos {
						sort.SliceStable(best, func(i, j int) bool {
							if best[i].pos != best[j].pos {
								return best[i].pos < best[j].pos
							}
							return best[i].idx < best[j].idx
						})
					}
					ids = ids[:0]
					for _, it := range best {
						ids = append(ids, it.sid)
					}
					break // prefer the first join table found
				}
			}
		}
		d.SchematicIDs = ids

		html, err := registry.LoadFiles(collectionsEditTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// CollectionsUpdateHandler handles POST updates to a collection (author-only).
func CollectionsUpdateHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
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
		slug := e.Request.PathValue("slug")
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
		if rec.GetString("author") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		title := e.Request.FormValue("title")
		if title == "" {
			title = e.Request.FormValue("name")
		}
		description := e.Request.FormValue("description")
		bannerURL := e.Request.FormValue("banner_url")
		if title != "" {
			rec.Set("title", title)
			rec.Set("name", title)
		}
		rec.Set("description", description)
		rec.Set("banner_url", bannerURL)
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to save collection")
		}
		dest := "/collections/" + slug
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}

// CollectionsDeleteHandler handles POST delete (soft-delete) for a collection (author-only).
func CollectionsDeleteHandler(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
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
		slug := e.Request.PathValue("slug")
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
		if rec.GetString("author") != e.Auth.Id {
			return e.String(http.StatusForbidden, "not allowed")
		}
		// Soft delete: set a string timestamp in "deleted" for compatibility with earlier filters
		rec.Set("deleted", time.Now().UTC().Format(time.RFC3339))
		if err := app.Save(rec); err != nil {
			return e.String(http.StatusInternalServerError, "failed to delete collection")
		}
		dest := "/collections"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", dest)
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, dest)
	}
}
