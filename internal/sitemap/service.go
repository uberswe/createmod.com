package sitemap

import (
	"fmt"
	"github.com/pocketbase/pocketbase"
	"github.com/sabloger/sitemap-generator/smg"
	"time"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

// Generate is used to generate sitemaps, should be called asynchronously on start and new page creation
func (*Service) Generate(app *pocketbase.PocketBase) {
	app.Logger().Info("sitemap generation started")
	schematics, err := app.FindRecordsByFilter("schematics", "1=1", "-created", -1, 0)
	if err != nil {
		app.Logger().Warn(err.Error())
	}
	users, err := app.FindRecordsByFilter("users", "1=1", "-created", -1, 0)
	if err != nil {
		app.Logger().Warn(err.Error())
	}
	now := time.Now().UTC()

	smi := smg.NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetSitemapIndexName("sitemap")
	smi.SetHostname("https://www.createmod.com")
	smi.SetOutputPath("template/dist/sitemaps/")

	smPages := smi.NewSitemap()
	smPages.SetName("pages")
	smPages.SetLastMod(&now)

	addPage(app, now, smPages, "/", 1.0, smg.Daily)
	addPage(app, now, smPages, "/about", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/upload", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/contact", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/guide", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/rules", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/terms-of-service", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/login", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/register", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/reset-password", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/news", 0.9, smg.Weekly)
	addPage(app, now, smPages, "/schematics", 0.9, smg.Daily)
	addPage(app, now, smPages, "/search", 0.9, smg.Daily)

	smSchematics := smi.NewSitemap()
	smSchematics.SetName("schematics")
	smSchematics.SetLastMod(&now)

	for i := range schematics {
		images := []*smg.SitemapImage{{ImageLoc: fmt.Sprintf("/api/files/%s/%s", schematics[i].BaseFilesPath(), schematics[i].GetString("featured_image"))}}
		for _, g := range schematics[i].GetStringSlice("gallery") {
			images = append(images, &smg.SitemapImage{ImageLoc: fmt.Sprintf("/api/files/%s/%s", schematics[i].BaseFilesPath(), g)})
		}
		err := smSchematics.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/schematics/%s", schematics[i].GetString("name")),
			LastMod:    &now,
			ChangeFreq: smg.Weekly,
			Priority:   0.8,
			Images:     images,
		})
		if err != nil {
			app.Logger().Error("Unable to add SitemapLoc:", "error", err)
		}
	}

	smUsers := smi.NewSitemap()
	smUsers.SetName("users")
	smUsers.SetLastMod(&now)

	for i := range users {
		images := make([]*smg.SitemapImage, 0)
		if users[i].GetString("avatar") != "" {
			images = append(images, &smg.SitemapImage{ImageLoc: fmt.Sprintf("/api/files/%s/%s", users[i].BaseFilesPath(), users[i].GetString("avatar"))})
		}
		err := smUsers.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/author/%s", users[i].GetString("username")),
			LastMod:    &now,
			ChangeFreq: smg.Weekly,
			Priority:   0.5,
			Images:     images,
		})
		if err != nil {
			app.Logger().Error("Unable to add SitemapLoc:", "error", err)
		}
	}

	// TODO add news

	filename, err := smi.Save()
	if err != nil {
		app.Logger().Error("Unable to Save Sitemap:", "error", err)
	}

	// TODO check if production site
	// Pings the Search engines. default Google and Bing, But you can add any other ping URL's
	// in this format: http://www.google.com/webmasters/tools/ping?sitemap=%s
	//err = smi.PingSearchEngines()
	//if err != nil {
	//	return
	//}

	app.Logger().Info("Sitemap generated", "filename", filename)

}

func addPage(app *pocketbase.PocketBase, now time.Time, pages *smg.Sitemap, s string, i float32, c smg.ChangeFreq) {
	err := pages.Add(&smg.SitemapLoc{
		Loc:        s,
		LastMod:    &now,
		ChangeFreq: c,
		Priority:   i,
	})
	if err != nil {
		app.Logger().Error("Unable to add SitemapLoc:", "error", err)
	}
}
