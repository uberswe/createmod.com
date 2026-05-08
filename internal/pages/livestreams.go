package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
)

var livestreamsTemplates = append([]string{
	"./template/livestreams.html",
}, commonTemplates...)

type LiveStream struct {
	Username    string
	UserID      string
	Title       string
	ViewerCount int
	ThumbnailURL string
	StreamURL    string
}

type LivestreamsData struct {
	DefaultData
	Streams []LiveStream
}

func LivestreamsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := LivestreamsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Live Streams")
		d.Description = i18n.T(d.Language, "page.livestreams.description")
		d.Slug = "/live"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Live Streams"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		// TODO: load from Redis cache populated by TwitchStreamSearchWorker
		d.Streams = []LiveStream{}

		if !isAuthenticated(e) {
			setPublicCacheControl(e, 60)
		}

		html, err := registry.LoadFiles(livestreamsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
