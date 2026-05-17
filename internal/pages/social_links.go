package pages

import (
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"net/url"
	"strings"
)

var allowedPlatforms = map[string]bool{
	"youtube":   true,
	"patreon":   true,
	"twitch":    true,
	"reddit":    true,
	"tiktok":    true,
	"instagram": true,
	"x":         true,
	"threads":   true,
	"bluesky":   true,
}

var platformDomains = map[string][]string{
	"youtube":   {"youtube.com", "youtu.be"},
	"patreon":   {"patreon.com"},
	"twitch":    {"twitch.tv"},
	"reddit":    {"reddit.com"},
	"tiktok":    {"tiktok.com"},
	"instagram": {"instagram.com"},
	"x":         {"x.com", "twitter.com"},
	"threads":   {"threads.net"},
	"bluesky":   {"bsky.app"},
}

func SocialLinkSaveHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)

		if err := e.Request.ParseForm(); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "Invalid form data"}
		}

		platform := strings.TrimSpace(e.Request.FormValue("platform"))
		linkURL := strings.TrimSpace(e.Request.FormValue("url"))
		username := strings.TrimSpace(e.Request.FormValue("username"))

		if !allowedPlatforms[platform] {
			return &server.APIError{Status: http.StatusBadRequest, Message: "Invalid platform"}
		}

		if linkURL != "" {
			parsed, err := url.Parse(linkURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return &server.APIError{Status: http.StatusBadRequest, Message: "Invalid URL"}
			}
			host := strings.TrimPrefix(strings.ToLower(parsed.Host), "www.")
			valid := false
			for _, d := range platformDomains[platform] {
				if host == d {
					valid = true
					break
				}
			}
			if !valid {
				return &server.APIError{Status: http.StatusBadRequest, Message: "URL does not match platform"}
			}
		}

		ctx := e.Request.Context()
		if err := appStore.SocialLinks.Upsert(ctx, &store.SocialLink{
			UserID:   userID,
			Platform: platform,
			URL:      linkURL,
			Username: username,
		}); err != nil {
			return err
		}

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/settings")
			return e.NoContent(http.StatusNoContent)
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/settings"))
	}
}

func SocialLinkDeleteHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)
		platform := e.Request.PathValue("platform")

		if !allowedPlatforms[platform] {
			return &server.APIError{Status: http.StatusBadRequest, Message: "Invalid platform"}
		}

		ctx := e.Request.Context()
		if err := appStore.SocialLinks.Delete(ctx, userID, platform); err != nil {
			return err
		}

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/settings")
			return e.NoContent(http.StatusNoContent)
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/settings"))
	}
}
