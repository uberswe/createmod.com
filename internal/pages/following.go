package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/translation"
	"net/http"
	"sort"
)

var followingTemplates = append([]string{
	"./template/following.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

type FollowingData struct {
	Schematics    []models.Schematic
	HasSchematics bool
	Follows       []store.UserFollow
	DefaultData
}

func FollowingHandler(cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		d := FollowingData{}
		d.Populate(e)
		d.Title = "Following"
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)

		ctx := e.Request.Context()
		userID := authenticatedUserID(e)

		follows, err := appStore.Follows.ListByUser(ctx, userID)
		if err == nil {
			d.Follows = follows
		}

		followedUserIDs, err := appStore.Follows.ListFollowedUserIDs(ctx, userID)
		if err != nil || len(followedUserIDs) == 0 {
			html, err := registry.LoadFiles(followingTemplates...).Render(d)
			if err != nil {
				return err
			}
			return e.HTML(http.StatusOK, html)
		}

		var allSchematics []models.Schematic
		for _, fid := range followedUserIDs {
			schematics := findAuthorSchematicsFromStore(appStore, cacheService, "", fid, 20)
			allSchematics = append(allSchematics, schematics...)
		}

		sort.Slice(allSchematics, func(i, j int) bool {
			return allSchematics[i].Created.After(allSchematics[j].Created)
		})

		if len(allSchematics) > 100 {
			allSchematics = allSchematics[:100]
		}

		translateSchematicTitles(allSchematics, translationService, cacheService, d.Language)

		d.Schematics = allSchematics
		d.HasSchematics = len(allSchematics) > 0

		html, err := registry.LoadFiles(followingTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
