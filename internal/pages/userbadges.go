package pages

import (
	"createmod/internal/cache"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strings"
)

var userBadgesTemplates = append([]string{
	"./template/user-badges.html",
}, commonTemplates...)

type UserBadgesData struct {
	AllBadges       []store.UserBadge
	DisplayedBadges []store.DisplayBadge
	HasBadges       bool
	DefaultData
}

func UserBadgesHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := UserBadgesData{}
		d.Populate(e)
		d.SettingsPage = "badges"
		d.Title = "Badges"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		userBadges, err := appStore.Badges.ListUserBadges(ctx, userID)
		if err == nil {
			d.AllBadges = userBadges
			d.HasBadges = len(userBadges) > 0
		}

		displayed, err := appStore.Badges.GetDisplayedBadges(ctx, userID)
		if err == nil {
			d.DisplayedBadges = displayed
		}

		html, err := registry.LoadFiles(userBadgesTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func UserBadgesSaveHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		userID := authenticatedUserID(e)

		if err := e.Request.ParseForm(); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "Invalid form data"}
		}

		badgeIDsRaw := e.Request.FormValue("displayed_badges")
		var badgeIDs []string
		if badgeIDsRaw != "" {
			parts := strings.Split(badgeIDsRaw, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					badgeIDs = append(badgeIDs, p)
				}
			}
		}
		if len(badgeIDs) > 3 {
			badgeIDs = badgeIDs[:3]
		}

		ctx := e.Request.Context()
		if err := appStore.Badges.SetDisplayedBadges(ctx, userID, badgeIDs); err != nil {
			return err
		}

		if e.Request.Header.Get("HX-Request") == "true" {
			e.Response.Header().Set("HX-Redirect", "/settings/badges")
			return e.NoContent(http.StatusNoContent)
		}
		return e.Redirect(http.StatusSeeOther, LangRedirectURL(e, "/settings/badges"))
	}
}
