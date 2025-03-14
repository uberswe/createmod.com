package pages

import (
	"createmod/internal/cache"
	"createmod/internal/models"
	"github.com/drexedam/gravatar"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"html/template"
	"strings"
	"time"
)

type DefaultData struct {
	IsAuthenticated bool
	Username        string
	UserID          string
	UsernameSlug    string
	Title           string
	Description     string
	Slug            string
	Thumbnail       string
	SubCategory     string
	Categories      []models.SchematicCategory
	Avatar          template.URL
	HasAvatar       bool
}

func (d *DefaultData) Populate(e *core.RequestEvent) {
	user := e.Auth
	if user != nil {
		d.IsAuthenticated = true
		caser := cases.Title(language.English)
		d.Username = caser.String(user.GetString("username"))
		d.UserID = user.Id
		d.UsernameSlug = strings.ToLower(user.GetString("username"))
		url := gravatar.New(user.GetString("email")).
			Size(200).
			Default(gravatar.MysteryMan).
			Rating(gravatar.Pg).
			AvatarURL()
		d.Avatar = template.URL(url)
		d.HasAvatar = d.Avatar != ""
	}
}

func allCategories(app *pocketbase.PocketBase, cacheService *cache.Service) []models.SchematicCategory {
	categories, found := cacheService.GetCategories(cache.AllCategoriesKey)
	if found {
		return categories
	}
	categoriesCollection, err := app.FindCollectionByNameOrId("schematic_categories")
	if err != nil {
		return nil
	}
	records, err := app.FindRecordsByFilter(categoriesCollection.Id, "1=1", "+key", -1, 0)
	if err != nil {
		return nil
	}
	categories = mapResultToCategories(records)
	// 730 hours = 1 month
	cacheService.SetCategories(cache.AllCategoriesKey, categories, time.Hour*730)
	return categories
}
