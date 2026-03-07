package pages

import (
	"context"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"strings"

	"createmod/internal/server"
)

// SchematicAddToCollectionHandler handles POST /schematics/{name}/add-to-collection
func SchematicAddToCollectionHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			return e.String(http.StatusMethodNotAllowed, "method not allowed")
		}
		name := e.Request.PathValue("name")
		returnTo := "/schematics/" + name
		if ok, err := requireAuth(e); !ok {
			return err
		}
		if err := e.Request.ParseForm(); err != nil {
			return e.String(http.StatusBadRequest, "invalid form")
		}
		collInput := e.Request.FormValue("collection")
		if collInput == "" {
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo+"?error=missing_collection"))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo+"?error=missing_collection"))
		}

		ctx := context.Background()

		// Resolve schematic by name
		schematic, err := appStore.Schematics.GetByName(ctx, name)
		if err != nil || schematic == nil {
			return e.String(http.StatusNotFound, "schematic not found")
		}

		var collectionID string

		if collInput == "__new__" {
			newName := strings.TrimSpace(e.Request.FormValue("new_collection_name"))
			if newName == "" {
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo+"?error=new_name_required"))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo+"?error=new_name_required"))
			}
			newColl := &store.Collection{
				Title: newName,
				Name:  newName,
			}
			authorID := authenticatedUserID(e)
			newColl.AuthorID = &authorID
			if err := appStore.Collections.Create(ctx, newColl); err != nil {
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo+"?error=failed_create_collection"))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo+"?error=failed_create_collection"))
			}
			collectionID = newColl.ID
		} else {
			// Try to find by slug first, then by ID
			coll, err := appStore.Collections.GetBySlug(ctx, collInput)
			if err != nil || coll == nil {
				coll, err = appStore.Collections.GetByID(ctx, collInput)
			}
			if coll == nil {
				if e.Request.Header.Get("HX-Request") != "" {
					e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo+"?error=collection_not_found"))
					return e.HTML(http.StatusNoContent, "")
				}
				return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo+"?error=collection_not_found"))
			}
			collectionID = coll.ID
		}

		// Add schematic to collection (AddSchematic handles duplicate prevention)
		if err := appStore.Collections.AddSchematic(ctx, collectionID, schematic.ID, 0); err != nil {
			slog.Warn("add-to-collection: failed", "error", err)
		}

		// If this was a newly created collection, redirect to the edit screen
		if collInput == "__new__" {
			dest := "/collections/" + collectionID + "/edit"
			if e.Request.Header.Get("HX-Request") != "" {
				e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, dest))
				return e.HTML(http.StatusNoContent, "")
			}
			return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, dest))
		}

		suffix := "?added=1"
		if e.Request.Header.Get("HX-Request") != "" {
			e.Response.Header().Set("HX-Redirect", LangRedirectURL(e, returnTo+suffix))
			return e.HTML(http.StatusNoContent, "")
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, returnTo+suffix))
	}
}
