package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strconv"
)

var topCreatorsTemplates = append([]string{
	"./template/top-creators.html",
}, commonTemplates...)

type TopCreatorsData struct {
	DefaultData
	Creators    []TopCreatorEntry
	Page        int
	TotalPages  int
	NextPageURL string
	PrevPageURL string
}

type TopCreatorEntry struct {
	Rank          int
	User          store.User
	DisplayBadges []store.DisplayBadge
	SocialLinks   []store.SocialLink
}

func TopCreatorsHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := TopCreatorsData{}
		d.Populate(e)
		d.Title = i18n.T(d.Language, "Top Creators")
		d.Description = i18n.T(d.Language, "page.topcreators.description")
		d.Slug = "/top-creators"
		d.Thumbnail = "https://createmod.com/assets/x/logo_sq_lg.png"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Top Creators"))
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		const perPage = 50
		page := 1
		if p := e.Request.URL.Query().Get("page"); p != "" {
			if pn, err := strconv.Atoi(p); err == nil && pn > 0 {
				page = pn
			}
		}
		d.Page = page
		offset := (page - 1) * perPage

		ctx := e.Request.Context()

		total, _ := appStore.Users.CountActive(ctx)
		d.TotalPages = int((total + int64(perPage) - 1) / int64(perPage))
		if d.TotalPages < 1 {
			d.TotalPages = 1
		}

		users, err := appStore.Users.ListTopByPoints(ctx, perPage, offset)
		if err != nil {
			return err
		}

		userIDs := make([]string, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}

		badgesMap, _ := appStore.Badges.BatchGetDisplayedBadges(ctx, userIDs)

		entries := make([]TopCreatorEntry, len(users))
		for i, u := range users {
			links, _ := appStore.SocialLinks.ListByUser(ctx, u.ID)
			entries[i] = TopCreatorEntry{
				Rank:          offset + i + 1,
				User:          u,
				DisplayBadges: badgesMap[u.ID],
				SocialLinks:   links,
			}
		}
		d.Creators = entries

		if page > 1 {
			d.PrevPageURL = "/top-creators?page=" + strconv.Itoa(page-1)
		}
		if page < d.TotalPages {
			d.NextPageURL = "/top-creators?page=" + strconv.Itoa(page+1)
		}

		if !isAuthenticated(e) {
			setPublicCacheControl(e, 300)
		}

		html, err := registry.LoadFiles(topCreatorsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
