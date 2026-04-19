package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/session"
	"createmod/internal/store"
	"createmod/internal/translation"
	"github.com/drexedam/gravatar"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	tmpl "html/template"
	"net/http"
	"sort"
)

// UserAchievement is a minimal UI struct for profile achievements.
type UserAchievement struct {
	Title       string
	Description string
	Icon        string
}

var profileTemplates = append([]string{
	"./template/profile.html",
	"./template/include/schematic_card.html",
	"./template/include/schematic_card_small.html",
}, commonTemplates...)

type ProfileData struct {
	ProfileUsername string
	Name            string
	HasSchematics  bool
	UserAvatar     tmpl.URL
	Schematics     []models.Schematic
	SchematicCount int
	TotalViews     int
	TotalDownloads int
	Points         int
	Sort           string
	// Achievements earned by this user (minimal display)
	Achievements    []UserAchievement
	HasAchievements bool
	DefaultData
}

func ProfileHandler(searchEngine search.SearchEngine, cacheService *cache.Service, registry *server.Registry, appStore *store.Store, translationService *translation.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			// /profile → redirect to the logged-in user's author page
			if u := session.UserFromContext(e.Request.Context()); u != nil {
				return e.Redirect(http.StatusFound, LangRedirectURL(e, "/author/"+u.Username))
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}
		return showProfile(e, searchEngine, appStore, cacheService, registry, translationService, username)
	}
}

func showProfile(e *server.RequestEvent, searchEngine search.SearchEngine, appStore *store.Store, cacheService *cache.Service, registry *server.Registry, translationService *translation.Service, username string) error {
	d := ProfileData{}
	d.Populate(e)
	caser := cases.Title(language.English)
	d.Breadcrumbs = NewBreadcrumbs(d.Language, caser.String(username))
	d.Title = i18n.T(d.Language, "Schematics by") + " " + caser.String(username)
	d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
	d.ProfileUsername = caser.String(username)
	d.Description = i18n.T(d.Language, "Find Create Mod schematics by") + " " + caser.String(username) + " " + i18n.T(d.Language, "on CreateMod.com")
	d.Slug = "/author/" + username

	ctx := e.Request.Context()
	user, err := appStore.Users.GetUserByUsername(ctx, username)
	if err != nil || user == nil || user.Deleted != nil {
		return RenderNotFound(registry, searchEngine, cacheService, appStore, e)
	}

	d.Schematics = findAuthorSchematicsFromStore(appStore, cacheService, "", user.ID, 1000)
	translateSchematicTitles(d.Schematics, translationService, cacheService, d.Language)
	if user.Avatar != "" {
		d.UserAvatar = tmpl.URL(user.Avatar)
		d.Thumbnail = user.Avatar
	} else {
		url := gravatar.New(user.Email).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		d.UserAvatar = tmpl.URL(url)
		d.Thumbnail = url
	}
	d.Points = user.Points

	// Usage stats
	d.SchematicCount = len(d.Schematics)
	totalViews := 0
	for _, s := range d.Schematics {
		totalViews += s.Views
	}
	d.TotalViews = totalViews

	// Sum downloads
	sum := 0
	for _, s := range d.Schematics {
		if cnt, err := appStore.ViewRatings.GetDownloadCount(ctx, s.ID); err == nil {
			sum += cnt
		}
	}
	d.TotalDownloads = sum

	// Load achievements
	achs, err := appStore.Achievements.ListUserAchievements(ctx, user.ID)
	if err == nil {
		uiAchs := make([]UserAchievement, 0, len(achs))
		for _, a := range achs {
			uiAchs = append(uiAchs, UserAchievement{
				Title:       a.Title,
				Description: a.Description,
				Icon:        a.Icon,
			})
		}
		d.Achievements = uiAchs
		d.HasAchievements = len(uiAchs) > 0
	}

	if len(d.Schematics) > 0 {
		d.HasSchematics = true
	}

	// Sort schematics based on query parameter
	sortParam := e.Request.URL.Query().Get("sort")
	switch sortParam {
	case "oldest":
		sort.Slice(d.Schematics, func(i, j int) bool {
			return d.Schematics[i].Created.Before(d.Schematics[j].Created)
		})
	case "views":
		sort.Slice(d.Schematics, func(i, j int) bool {
			return d.Schematics[i].Views > d.Schematics[j].Views
		})
	case "downloads":
		sort.Slice(d.Schematics, func(i, j int) bool {
			return d.Schematics[i].Downloads > d.Schematics[j].Downloads
		})
	default:
		sortParam = "recent"
		sort.Slice(d.Schematics, func(i, j int) bool {
			return d.Schematics[i].Created.After(d.Schematics[j].Created)
		})
	}
	d.Sort = sortParam

	html, err := registry.LoadFiles(profileTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

