package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

var userSubscriptionsTemplates = append([]string{
	"./template/user-subscriptions.html",
}, commonTemplates...)

type UserSubscriptionsData struct {
	DefaultData
	SearchAlerts         []store.SearchAlert
	SectionSubscriptions []store.SectionSubscription
}

func UserSubscriptionsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := UserSubscriptionsData{}
		d.Populate(e)
		d.SettingsPage = "subscriptions"
		d.HideOutstream = true
		d.Title = i18n.T(d.Language, "Subscriptions")
		d.Slug = "/settings/subscriptions"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Settings"), "/settings", i18n.T(d.Language, "Subscriptions"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		alerts, err := appStore.SearchAlerts.ListByUser(ctx, userID)
		if err == nil {
			d.SearchAlerts = alerts
		}

		subs, err := appStore.SectionSubscriptions.ListByUser(ctx, userID)
		if err == nil {
			d.SectionSubscriptions = subs
		}

		html, err := registry.LoadFiles(userSubscriptionsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func DeleteSearchAlertHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)
		alertID := e.Request.PathValue("id")

		if err := appStore.SearchAlerts.Delete(ctx, alertID, userID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to delete alert"}
		}

		return e.String(http.StatusOK, "")
	}
}

func DeleteSectionSubscriptionHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)
		subID := e.Request.PathValue("id")

		if err := appStore.SectionSubscriptions.Delete(ctx, subID, userID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to delete subscription"}
		}

		return e.String(http.StatusOK, "")
	}
}
