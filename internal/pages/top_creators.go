package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/server"
	"createmod/internal/store"
	"net/http"
	"strconv"

	"github.com/drexedam/gravatar"
)

var topCreatorsTemplates = append([]string{
	"./template/top-creators.html",
}, commonTemplates...)

type TopCreatorsData struct {
	DefaultData
	CreatorsLeft  []TopCreatorEntry
	CreatorsRight []TopCreatorEntry
	HowToEarn     []HowToEarnItem
	Page          int
	TotalPages    int
	NextPageURL   string
	PrevPageURL   string
	CurrentUser   *TopCreatorEntry
	CurrentRank   int
}

type TopCreatorEntry struct {
	Rank          int
	User          store.User
	AvatarURL     string
	DisplayBadges []store.DisplayBadge
	SocialLinks   []store.SocialLink
}

func avatarURLForUser(u store.User) string {
	if u.Avatar != "" {
		return u.Avatar
	}
	if u.Email != "" {
		return gravatar.New(u.Email).
			Size(80).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
	}
	return "https://www.gravatar.com/avatar/?d=mp&s=80"
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

		const perPage = 100
		const splitAt = 60
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
				AvatarURL:     avatarURLForUser(u),
				DisplayBadges: badgesMap[u.ID],
				SocialLinks:   links,
			}
		}

		if len(entries) > splitAt {
			d.CreatorsLeft = entries[:splitAt]
			d.CreatorsRight = entries[splitAt:]
		} else {
			d.CreatorsLeft = entries
		}

		d.HowToEarn = howToEarnItems(d.Language)

		if page > 1 {
			d.PrevPageURL = "/top-creators?page=" + strconv.Itoa(page-1)
		}
		if page < d.TotalPages {
			d.NextPageURL = "/top-creators?page=" + strconv.Itoa(page+1)
		}

		if isAuthenticated(e) {
			uid := authenticatedUserID(e)
			if u, err := appStore.Users.GetUserByID(ctx, uid); err == nil {
				rank, _ := appStore.Users.GetUserPointsRank(ctx, uid)
				d.CurrentRank = int(rank)
				d.CurrentUser = &TopCreatorEntry{
					Rank:      int(rank),
					User:      *u,
					AvatarURL: avatarURLForUser(*u),
				}
			}
		} else {
			setPublicCacheControl(e, 300)
		}

		html, err := registry.LoadFiles(topCreatorsTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
