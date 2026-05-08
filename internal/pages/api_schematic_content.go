package pages

import (
	"context"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"net/url"
	"strings"
)

// AddSchematicVideoHandler handles POST /api/schematics/{id}/videos.
func AddSchematicVideoHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		videoURL := strings.TrimSpace(e.Request.FormValue("video_url"))
		if videoURL == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "video_url is required"}
		}
		if !IsValidYouTubeVideo(videoURL) {
			return &server.APIError{Status: http.StatusBadRequest, Message: "video must be a valid YouTube link"}
		}

		videoType := strings.TrimSpace(e.Request.FormValue("video_type"))
		if videoType == "" {
			videoType = "showcase"
		}
		title := strings.TrimSpace(e.Request.FormValue("title"))

		video := store.SchematicVideo{
			SchematicID: schematicID,
			VideoURL:    videoURL,
			VideoType:   videoType,
			Title:       title,
		}
		if err := appStore.SchematicVideos.Create(ctx, &video); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to add video"}
		}

		return e.JSON(http.StatusOK, map[string]string{"id": video.ID})
	}
}

// DeleteSchematicVideoHandler handles DELETE /api/schematics/{id}/videos/{videoId}.
func DeleteSchematicVideoHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		videoID := e.Request.PathValue("videoId")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		if err := appStore.SchematicVideos.Delete(ctx, videoID, schematicID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to delete video"}
		}

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}

var allowedReferenceDomains = map[string]bool{
	"createmod.com":             true,
	"www.reddit.com":            true,
	"reddit.com":                true,
	"youtube.com":               true,
	"www.youtube.com":           true,
	"youtu.be":                  true,
	"schematicannon.com":        true,
	"www.schematicannon.com":    true,
	"abfielder.com":             true,
	"www.abfielder.com":         true,
	"minecraft-schematics.net":  true,
	"planetminecraft.com":       true,
	"www.planetminecraft.com":   true,
}

// AddSchematicReferenceHandler handles POST /api/schematics/{id}/references.
func AddSchematicReferenceHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		refURL := strings.TrimSpace(e.Request.FormValue("url"))
		if refURL == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "url is required"}
		}
		parsed, pErr := url.Parse(refURL)
		if pErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid URL"}
		}
		host := strings.ToLower(parsed.Hostname())
		if !allowedReferenceDomains[host] {
			return &server.APIError{Status: http.StatusBadRequest, Message: "domain not allowed for references"}
		}

		ref := store.SchematicReference{
			SchematicID: schematicID,
			URL:         refURL,
			Title:       strings.TrimSpace(e.Request.FormValue("title")),
		}
		if err := appStore.References.Create(ctx, &ref); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to add reference"}
		}

		return e.JSON(http.StatusOK, map[string]string{"id": ref.ID})
	}
}

// DeleteSchematicReferenceHandler handles DELETE /api/schematics/{id}/references/{refId}.
func DeleteSchematicReferenceHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		refID := e.Request.PathValue("refId")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		if err := appStore.References.Delete(ctx, refID, schematicID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to delete reference"}
		}

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}

var allowedRedditSubreddits = map[string]bool{
	"createmod":          true,
	"minecraft":          true,
	"feedthebeast":       true,
	"createmodshowcase":  true,
	"moddedminecraft":    true,
}

// AddSchematicRedditLinkHandler handles POST /api/schematics/{id}/reddit-links.
func AddSchematicRedditLinkHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		redditURL := strings.TrimSpace(e.Request.FormValue("reddit_url"))
		if redditURL == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "reddit_url is required"}
		}
		parsed, pErr := url.Parse(redditURL)
		if pErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid URL"}
		}
		host := strings.ToLower(parsed.Hostname())
		if host != "reddit.com" && host != "www.reddit.com" && host != "old.reddit.com" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "URL must be a reddit.com link"}
		}

		// Extract subreddit from path: /r/{subreddit}/...
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		subreddit := ""
		for i, p := range parts {
			if p == "r" && i+1 < len(parts) {
				subreddit = strings.ToLower(parts[i+1])
				break
			}
		}
		if subreddit == "" || !allowedRedditSubreddits[subreddit] {
			return &server.APIError{Status: http.StatusBadRequest, Message: "subreddit not allowed"}
		}

		link := store.RedditLink{
			SchematicID: schematicID,
			RedditURL:   redditURL,
			Subreddit:   subreddit,
		}
		if err := appStore.RedditLinks.Create(ctx, &link); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to add Reddit link"}
		}

		return e.JSON(http.StatusOK, map[string]string{"id": link.ID})
	}
}

// DeleteSchematicRedditLinkHandler handles DELETE /api/schematics/{id}/reddit-links/{linkId}.
func DeleteSchematicRedditLinkHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		linkID := e.Request.PathValue("linkId")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		if err := appStore.RedditLinks.Delete(ctx, linkID, schematicID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to delete Reddit link"}
		}

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}

// SetSchematicModpacksHandler handles POST /api/schematics/{id}/modpacks.
func SetSchematicModpacksHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		schematicID := e.Request.PathValue("id")
		ctx := context.Background()

		schem, err := appStore.Schematics.GetByID(ctx, schematicID)
		if err != nil || schem == nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "schematic not found"}
		}
		if schem.AuthorID != authenticatedUserID(e) {
			return &server.APIError{Status: http.StatusForbidden, Message: "not authorized"}
		}

		if err := e.Request.ParseForm(); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid form data"}
		}
		modpackIDs := e.Request.Form["modpack_ids"]
		if err := appStore.Modpacks.SetSchematicModpacks(ctx, schematicID, modpackIDs); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to update modpacks"}
		}

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}
