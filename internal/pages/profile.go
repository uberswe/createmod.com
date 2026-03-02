package pages

import (
	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/session"
	"createmod/internal/store"
	"github.com/drexedam/gravatar"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	tmpl "html/template"
	"net/http"
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
	Username       string
	Name           string
	HasSchematics  bool
	UserAvatar     tmpl.URL
	Schematics     []models.Schematic
	SchematicCount int
	TotalViews     int
	TotalDownloads int
	Points int
	// Achievements earned by this user (minimal display)
	Achievements    []UserAchievement
	HasAchievements bool
	DefaultData
}

func ProfileHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			if u := session.UserFromContext(e.Request.Context()); u != nil {
				return showProfile(e, app, appStore, cacheService, registry, u.Username)
			}
			return e.Redirect(http.StatusFound, LangRedirectURL(e, "/login"))
		}
		return showProfile(e, app, appStore, cacheService, registry, username)
	}
}

func showProfile(e *core.RequestEvent, app *pocketbase.PocketBase, appStore *store.Store, cacheService *cache.Service, registry *template.Registry, username string) error {
	d := ProfileData{}
	d.Populate(e)
	caser := cases.Title(language.English)
	d.Title = i18n.T(d.Language, "Schematics by") + " " + caser.String(username)
	d.Categories = allCategoriesFromStore(appStore, app, cacheService)
	d.Username = caser.String(username)
	d.Description = i18n.T(d.Language, "Find Create Mod schematics by") + " " + caser.String(username) + " " + i18n.T(d.Language, "on CreateMod.com")
	d.Slug = "/author/" + username

	ctx := e.Request.Context()
	user, err := appStore.Users.GetUserByUsername(ctx, username)
	if err != nil || user == nil || user.Deleted != nil {
		return e.HTML(http.StatusNotFound, "User not found")
	}

	d.Schematics = findAuthorSchematics(app, cacheService, "", user.ID, 1000, "-created")
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

	html, err := registry.LoadFiles(profileTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}

