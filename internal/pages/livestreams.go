package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"fmt"
	"net/http"
	"strings"
)

var livestreamsTemplates = append([]string{
	"./template/livestreams.html",
}, commonTemplates...)

type LiveStream struct {
	Username     string
	Title        string
	ViewerCount  int
	ThumbnailURL string
	StreamURL    string
	IsSiteMember bool
}

type LivestreamsData struct {
	DefaultData
	Streams []LiveStream
}

func LivestreamsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if redirected, err := RedirectToPreferredLang(e); redirected || err != nil {
			return err
		}
		d := LivestreamsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Live Streams")
		d.Description = i18n.T(d.Language, "page.livestreams.description")
		d.Slug = "/live"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Live Streams"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		if cached, found := cacheService.Get("twitch_live_streams"); found {
			if streams, ok := cached.([]store.CachedTwitchStream); ok {
				var siteMembers map[string]bool
				if sm, found := cacheService.Get("twitch_site_members"); found {
					if m, ok := sm.(map[string]bool); ok {
						siteMembers = m
					}
				}
				if siteMembers == nil {
					siteMembers = map[string]bool{}
				}

				for _, s := range streams {
					ls := LiveStream{
						Username:     s.UserName,
						Title:        s.Title,
						ViewerCount:  s.ViewerCount,
						ThumbnailURL: s.ThumbnailURL,
						StreamURL:    fmt.Sprintf("https://www.twitch.tv/%s", s.UserLogin),
						IsSiteMember: siteMembers[strings.ToLower(s.UserLogin)],
					}
					d.Streams = append(d.Streams, ls)
				}
				sortStreams(d.Streams)
			}
		}

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

func sortStreams(streams []LiveStream) {
	for i := 0; i < len(streams); i++ {
		for j := i + 1; j < len(streams); j++ {
			swap := false
			if streams[j].IsSiteMember && !streams[i].IsSiteMember {
				swap = true
			} else if streams[j].IsSiteMember == streams[i].IsSiteMember && streams[j].ViewerCount > streams[i].ViewerCount {
				swap = true
			}
			if swap {
				streams[i], streams[j] = streams[j], streams[i]
			}
		}
	}
}
