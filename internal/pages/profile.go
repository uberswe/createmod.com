package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/drexedam/gravatar"
	"github.com/pocketbase/dbx"
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

func ProfileHandler(app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		username := e.Request.PathValue("username")
		if username == "" {
			return editProfile(e, app, registry, cacheService)
		}
		return showProfile(e, app, cacheService, registry, username)
	}
}

func showProfile(e *core.RequestEvent, app *pocketbase.PocketBase, cacheService *cache.Service, registry *template.Registry, username string) error {
	d := ProfileData{}
	d.Populate(e)
	caser := cases.Title(language.English)
	d.Title = "Schematics by " + caser.String(username)
	d.Categories = allCategories(app, cacheService)
	d.Username = caser.String(username)
	d.Description = "Find Create Mod schematics by " + caser.String(username) + " on CreateMod.com"
	d.Slug = "/author/" + username

	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	results, err := app.FindRecordsByFilter(
		usersCollection.Id,
		"username:lower = {:username} && deleted = ''",
		"-created",
		1,
		0,
		dbx.Params{"username": e.Request.PathValue("username")})

	if err != nil {
		return err
	}

	if len(results) == 1 {
		d.Schematics = findAuthorSchematics(app, cacheService, "", results[0].Id, 1000, "-created")
		url := gravatar.New(results[0].GetString("email")).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		d.UserAvatar = tmpl.URL(url)
		d.Thumbnail = url
		// Load points
		d.Points = results[0].GetInt("points")

		// Usage stats
		d.SchematicCount = len(d.Schematics)
		totalViews := 0
		for _, s := range d.Schematics {
			totalViews += s.Views
		}
		d.TotalViews = totalViews
		// Sum downloads (type=4, period="total") best-effort
		if coll, err := app.FindCollectionByNameOrId("schematic_downloads"); err == nil && coll != nil {
			sum := 0
			for _, s := range d.Schematics {
				recs, _ := app.FindRecordsByFilter(coll.Id, "schematic = {:schematic} && type = {:type} && period = {:period}", "-created", 1, 0, dbx.Params{"schematic": s.ID, "type": 4, "period": "total"})
				if len(recs) > 0 {
					sum += recs[0].GetInt("count")
				}
			}
			d.TotalDownloads = sum
		}

		// Load achievements
		if uaColl, err := app.FindCollectionByNameOrId("user_achievements"); err == nil && uaColl != nil {
			if achColl, err := app.FindCollectionByNameOrId("achievements"); err == nil && achColl != nil {
				if uas, err := app.FindRecordsByFilter(uaColl.Id, "user = {:u}", "-created", 100, 0, dbx.Params{"u": results[0].Id}); err == nil {
					achs := make([]UserAchievement, 0, len(uas))
					for _, ua := range uas {
						achID := ua.GetString("achievement")
						if achID == "" {
							continue
						}
						if rec, err := app.FindRecordById(achColl.Id, achID); err == nil && rec != nil {
							achs = append(achs, UserAchievement{
								Title:       rec.GetString("title"),
								Description: rec.GetString("description"),
								Icon:        rec.GetString("icon"),
							})
						}
					}
					d.Achievements = achs
					d.HasAchievements = len(achs) > 0
				}
			}
		}
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

func editProfile(e *core.RequestEvent, app *pocketbase.PocketBase, registry *template.Registry, cacheService *cache.Service) error {
	// TODO make this possible as part of #51
	d := ProfileData{}
	d.Populate(e)
	d.Title = "Edit profile coming soon"
	d.Categories = allCategories(app, cacheService)
	html, err := registry.LoadFiles(profileTemplates...).Render(d)
	if err != nil {
		return err
	}
	return e.HTML(http.StatusOK, html)
}
