package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/server"
	"createmod/internal/store"
	"createmod/internal/translation"
	"net/http"
	"sort"
	"strings"
)

var followingTemplates = append([]string{
	"./template/following.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

type FollowEntry struct {
	store.UserFollow
	DisplayName string
	Link        string
}

type FollowingData struct {
	Schematics    []models.Schematic
	HasSchematics bool
	Follows       []FollowEntry
	DefaultData
}

func FollowingUnfollowHandler(appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if ok, err := requireAuth(e); !ok {
			return err
		}
		ctx := e.Request.Context()
		userID := authenticatedUserID(e)
		followType := e.Request.FormValue("follow_type")
		targetID := e.Request.FormValue("target_id")
		if followType == "" || targetID == "" {
			return &server.APIError{Status: http.StatusBadRequest, Message: "follow_type and target_id are required"}
		}
		_ = appStore.Follows.Unfollow(ctx, userID, followType, targetID)
		return e.HTML(http.StatusOK, "")
	}
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
			d.Follows = make([]FollowEntry, 0, len(follows))
			for _, f := range follows {
				entry := FollowEntry{UserFollow: f}
				switch f.FollowType {
				case "user":
					if u, err := appStore.Users.GetUserByID(ctx, f.TargetID); err == nil {
						entry.DisplayName = u.Username
						entry.Link = "/author/" + strings.ToLower(u.Username)
					}
				case "category":
					if c, err := appStore.Categories.GetByID(ctx, f.TargetID); err == nil {
						entry.DisplayName = c.Name
						entry.Link = "/category/" + c.Key
					}
				case "search":
					entry.DisplayName = f.TargetID
					entry.Link = "/search?q=" + f.TargetID
				default:
					entry.DisplayName = f.TargetID
				}
				if entry.DisplayName == "" {
					entry.DisplayName = f.TargetID
				}
				d.Follows = append(d.Follows, entry)
			}
		}

		var allSchematics []models.Schematic
		seen := make(map[string]bool)
		addUnique := func(items []models.Schematic, limit int) {
			for _, s := range items {
				if seen[s.ID] {
					continue
				}
				seen[s.ID] = true
				allSchematics = append(allSchematics, s)
				if limit > 0 && len(allSchematics) >= limit {
					return
				}
			}
		}

		for _, f := range follows {
			switch f.FollowType {
			case "user":
				schematics := findAuthorSchematicsFromStore(appStore, cacheService, "", f.TargetID, 20)
				addUnique(schematics, 0)
			case "category":
				catSchematics, err := appStore.Schematics.ListByCategoryIDs(ctx, []string{f.TargetID}, nil, 16)
				if err == nil {
					addUnique(MapStoreSchematics(appStore, catSchematics, cacheService), 0)
				}
			case "trending":
				trending := getAllTrendingSchematicsForWindow(appStore, cacheService, 30)
				if len(trending) > 16 {
					trending = trending[:16]
				}
				addUnique(trending, 0)
			case "latest":
				if latest, ok := cacheService.GetSchematics(cache.LatestSchematicsKey); ok {
					if len(latest) > 16 {
						latest = latest[:16]
					}
					addUnique(latest, 0)
				}
			case "highest_rated":
				if rated, ok := cacheService.GetSchematics(cache.HighestRatedSchematicsKey); ok {
					if len(rated) > 16 {
						rated = rated[:16]
					}
					addUnique(rated, 0)
				}
			}
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
