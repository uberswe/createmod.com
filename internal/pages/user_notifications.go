package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

var userNotificationsTemplates = append([]string{
	"./template/user-notifications.html",
}, commonTemplates...)

type UserNotificationsData struct {
	DefaultData
	Preferences []store.NotificationPreference
}

func UserNotificationsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserNotificationsData{}
		d.Populate(e)
		d.SettingsPage = "notifications"
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Notification Settings")
		d.Slug = "/settings/notifications"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Notifications"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		prefs, err := appStore.Notifications.GetPreferences(ctx, userID)
		if err == nil {
			d.Preferences = prefs
		}

		html, err := registry.LoadFiles(userNotificationsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func UserNotificationsSaveHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		if err := e.Request.ParseForm(); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid form"}
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		categories := []string{"rating", "comment", "follow", "badge", "moderation"}
		for _, cat := range categories {
			email := e.Request.Form.Get("email_" + cat)
			if email == "" {
				email = "off"
			}
			web := e.Request.Form.Get("web_"+cat) == "on"

			_ = appStore.Notifications.UpsertPreference(ctx, &store.NotificationPreference{
				UserID:   userID,
				Category: cat,
				Email:    email,
				Web:      web,
			})
		}

		return e.Redirect(http.StatusSeeOther, "/settings/notifications")
	}
}
