package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

var unsubscribeTemplates = append([]string{
	"./template/unsubscribe.html",
}, commonTemplates...)

var newsletterViewTemplates = append([]string{
	"./template/newsletter-view.html",
}, commonTemplates...)

type UnsubscribeData struct {
	DefaultData
	Success bool
}

type NewsletterViewData struct {
	DefaultData
	Issue store.NewsletterIssue
}

func NewsletterSubscribeHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseForm(); err != nil {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid form"}
		}

		email := strings.TrimSpace(e.Request.Form.Get("email"))
		if email == "" || !strings.Contains(email, "@") {
			return &server.APIError{Status: http.StatusBadRequest, Message: "invalid email"}
		}

		confirmBuf := make([]byte, 16)
		_, _ = rand.Read(confirmBuf)
		unsubBuf := make([]byte, 16)
		_, _ = rand.Read(unsubBuf)

		ctx := e.Request.Context()
		var userID *string
		if uid := authenticatedUserID(e); uid != "" {
			userID = &uid
		}

		_ = appStore.Newsletters.Subscribe(ctx, &store.NewsletterSubscriber{
			Email:            email,
			UserID:           userID,
			Type:             "trending",
			Frequency:        "weekly",
			ConfirmToken:     hex.EncodeToString(confirmBuf),
			UnsubscribeToken: hex.EncodeToString(unsubBuf),
		})

		if e.Request.Header.Get("HX-Request") != "" {
			return e.HTML(http.StatusOK, `<p class="text-success small mb-0">&#10003; Subscribed! Check your inbox to confirm.</p>`)
		}
		return e.String(http.StatusOK, "subscribed")
	}
}

func UnsubscribeHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := UnsubscribeData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Unsubscribe")
		d.Slug = "/unsubscribe"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		token := e.Request.URL.Query().Get("token")
		if token != "" {
			ctx := e.Request.Context()
			_ = appStore.Newsletters.Unsubscribe(ctx, token)
			_ = appStore.SearchAlerts.Unsubscribe(ctx, token)
			_ = appStore.Follows.Unsubscribe(ctx, token)
			d.Success = true
		}

		html, err := registry.LoadFiles(unsubscribeTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

func NewsletterViewHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		slug := e.Request.PathValue("slug")

		ctx := e.Request.Context()
		issue, err := appStore.Newsletters.GetIssueBySlug(ctx, slug)
		if err != nil {
			return &server.APIError{Status: http.StatusNotFound, Message: "newsletter not found"}
		}

		d := NewsletterViewData{}
		d.Populate(e)
		d.Title = issue.Subject
		d.Slug = "/newsletters/" + slug
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Issue = *issue

		html, err := registry.LoadFiles(newsletterViewTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
