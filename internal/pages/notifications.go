package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strconv"
)

var notificationsTemplates = append([]string{
	"./template/notifications.html",
}, commonTemplates...)

type NotificationsData struct {
	DefaultData
	Notifications []store.Notification
	Page          int
	HasMore       bool
}

func NotificationsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		d := NotificationsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Notifications")
		d.Description = i18n.T(d.Language, "page.notifications.description")
		d.Slug = "/notifications"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Notifications"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		const perPage = 20
		page := 1
		if p := e.Request.URL.Query().Get("page"); p != "" {
			if pn, err := strconv.Atoi(p); err == nil && pn > 0 {
				page = pn
			}
		}
		d.Page = page
		offset := (page - 1) * perPage

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		notifs, err := appStore.Notifications.ListByUser(ctx, userID, perPage+1, offset)
		if err != nil {
			return err
		}

		if len(notifs) > perPage {
			d.HasMore = true
			notifs = notifs[:perPage]
		}
		d.Notifications = notifs

		html, err := registry.LoadFiles(notificationsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func NotificationsRecentHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		notifs, err := appStore.Notifications.ListRecent(ctx, userID, 10)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load notifications"})
		}

		return e.JSON(http.StatusOK, notifs)
	}
}

func NotificationsUnreadCountHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		userID := authenticatedUserID(e)
		if userID == "" {
			return e.HTML(http.StatusOK, "")
		}
		ctx := e.Request.Context()
		count, err := appStore.Notifications.CountUnread(ctx, userID)
		if err != nil || count == 0 {
			return e.HTML(http.StatusOK, "")
		}
		label := strconv.Itoa(count)
		if count > 99 {
			label = "99+"
		}
		return e.HTML(http.StatusOK, `<span class="badge bg-red badge-notification badge-blink">`+label+`</span>`)
	}
}

func NotificationsMarkReadHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		if err := appStore.Notifications.MarkAllRead(ctx, userID); err != nil {
			return &server.APIError{Status: http.StatusInternalServerError, Message: "failed to mark notifications read"}
		}

		return e.String(http.StatusOK, "ok")
	}
}
