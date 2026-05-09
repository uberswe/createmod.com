package pages

import (
	"encoding/json"
	"net/http"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
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

func CreateSearchAlertHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		var body struct {
			Query     string          `json:"query"`
			Filters   json.RawMessage `json:"filters"`
			Frequency string          `json:"frequency"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid request body"}
		}
		if body.Query == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "query is required"}
		}
		if body.Frequency == "" {
			body.Frequency = "daily"
		}

		alert := &store.SearchAlert{
			UserID:           userID,
			Query:            body.Query,
			Filters:          body.Filters,
			Frequency:        body.Frequency,
			Active:           true,
			UnsubscribeToken: randomHex(16),
		}
		if err := appStore.SearchAlerts.Create(ctx, alert); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to create alert"}
		}

		return e.JSON(http.StatusCreated, map[string]string{"id": alert.ID})
	}
}

func CreateSectionSubscriptionHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		var body struct {
			SubscriptionType string `json:"subscription_type"`
			TargetID         string `json:"target_id"`
			Frequency        string `json:"frequency"`
		}
		if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid request body"}
		}
		if body.SubscriptionType == "" || body.TargetID == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "subscription_type and target_id are required"}
		}
		if body.Frequency == "" {
			body.Frequency = "daily"
		}

		sub := &store.SectionSubscription{
			UserID:           userID,
			SubscriptionType: body.SubscriptionType,
			TargetID:         body.TargetID,
			Frequency:        body.Frequency,
			UnsubscribeToken: randomHex(16),
		}
		if err := appStore.SectionSubscriptions.Create(ctx, sub); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to create subscription"}
		}

		return e.JSON(http.StatusCreated, map[string]string{"id": sub.ID})
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
