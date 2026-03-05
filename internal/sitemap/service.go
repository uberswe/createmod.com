package sitemap

import (
	"context"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"time"

	"github.com/sabloger/sitemap-generator/smg"
)

type Service struct{ dev bool }

func New(dev bool) *Service {
	return &Service{dev: dev}
}

// Generate is used to generate sitemaps, should be called asynchronously on start and new page creation
func (s *Service) Generate(appStore *store.Store) {
	slog.Info("sitemap generation started")
	ctx := context.Background()

	schematics, err := appStore.Schematics.ListForSitemap(ctx)
	if err != nil {
		slog.Warn("sitemap: failed to list schematics", "error", err)
	}
	users, err := appStore.Users.ListForSitemap(ctx)
	if err != nil {
		slog.Warn("sitemap: failed to list users", "error", err)
	}
	searches, err := appStore.SearchTracking.ListTopSearches(ctx, 1000)
	if err != nil {
		slog.Warn("sitemap: failed to list searches", "error", err)
	}
	now := time.Now().UTC()

	smi := smg.NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetSitemapIndexName("sitemap")
	smi.SetHostname("https://www.createmod.com")
	smi.SetOutputPath("template/dist/sitemaps/")
	smi.SetServerURI("/sitemaps/")

	smPages := smi.NewSitemap()
	smPages.SetName("pages")
	smPages.SetLastMod(&now)
	smPages.SetOutputPath("template/dist/sitemaps/")

	addPage(now, smPages, "/", 1.0, smg.Daily)
	addPage(now, smPages, "/upload", 0.9, smg.Weekly)
	addPage(now, smPages, "/contact", 0.9, smg.Weekly)
	addPage(now, smPages, "/guide", 0.9, smg.Weekly)
	addPage(now, smPages, "/rules", 0.9, smg.Weekly)
	addPage(now, smPages, "/terms-of-service", 0.9, smg.Weekly)
	addPage(now, smPages, "/privacy-policy", 0.9, smg.Weekly)
	addPage(now, smPages, "/login", 0.9, smg.Weekly)
	addPage(now, smPages, "/register", 0.9, smg.Weekly)
	addPage(now, smPages, "/reset-password", 0.9, smg.Weekly)
	addPage(now, smPages, "/news", 0.9, smg.Weekly)
	addPage(now, smPages, "/schematics", 0.9, smg.Daily)
	addPage(now, smPages, "/search", 0.9, smg.Daily)
	addPage(now, smPages, "/explore", 0.9, smg.Daily)
	addPage(now, smPages, "/users", 0.6, smg.Weekly)
	addPage(now, smPages, "/videos", 0.6, smg.Weekly)

	schematicsSmCount := 1
	smSchematics := smi.NewSitemap()
	smSchematics.SetName(fmt.Sprintf("schematics-%d", schematicsSmCount))
	smSchematics.SetOutputPath("template/dist/sitemaps/")
	smSchematics.SetLastMod(&now)

	for i := range schematics {
		if i != 0 && i%1000 == 0 {
			schematicsSmCount++
			smSchematics = smi.NewSitemap()
			smSchematics.SetName(fmt.Sprintf("schematics-%d", schematicsSmCount))
			smSchematics.SetOutputPath("template/dist/sitemaps/")
			smSchematics.SetLastMod(&now)
		}
		err := smSchematics.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/schematics/%s", schematics[i].Name),
			LastMod:    &now,
			ChangeFreq: smg.Weekly,
			Priority:   0.8,
		})
		if err != nil {
			slog.Error("Unable to add SitemapLoc:", "error", err)
		}
	}

	userSmCount := 1
	smUsers := smi.NewSitemap()
	smUsers.SetName(fmt.Sprintf("users-%d", userSmCount))
	smUsers.SetLastMod(&now)
	smUsers.SetOutputPath("template/dist/sitemaps/")
	for i := range users {
		if i != 0 && i%1000 == 0 {
			userSmCount++
			smUsers = smi.NewSitemap()
			smUsers.SetName(fmt.Sprintf("users-%d", userSmCount))
			smUsers.SetLastMod(&now)
			smUsers.SetOutputPath("template/dist/sitemaps/")
		}
		err := smUsers.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/author/%s", users[i].Username),
			LastMod:    &now,
			ChangeFreq: smg.Weekly,
			Priority:   0.5,
		})
		if err != nil {
			slog.Error("Unable to add SitemapLoc:", "error", err)
		}
	}

	smSearches := smi.NewSitemap()
	smSearches.SetName("searches")
	smSearches.SetLastMod(&now)
	smSearches.SetOutputPath("template/dist/sitemaps/")

	for i := range searches {
		err := smSearches.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/search/%s", searches[i].Query),
			LastMod:    &now,
			ChangeFreq: smg.Weekly,
			Priority:   0.7,
		})
		if err != nil {
			slog.Error("Unable to add SitemapLoc:", "error", err)
		}
	}

	// TODO: add news sitemap via store
	smNews := smi.NewSitemap()
	smNews.SetName("news")
	smNews.SetLastMod(&now)
	smNews.SetOutputPath("template/dist/sitemaps/")
	if err := smNews.Add(&smg.SitemapLoc{
		Loc:        "/news",
		LastMod:    &now,
		ChangeFreq: smg.Weekly,
		Priority:   0.6,
	}); err != nil {
		slog.Error("Unable to add SitemapLoc:", "error", err)
	}

	filename, err := smi.Save()
	if err != nil {
		slog.Error("Unable to Save Sitemap:", "error", err)
	}

	// Only ping search engines in production
	if !s.dev {
		if err := smi.PingSearchEngines(); err != nil {
			slog.Warn("PingSearchEngines failed", "error", err)
		}
	}

	slog.Info("Sitemap generated", "filename", filename)

	// Generate hreflang sitemaps with xhtml:link alternates for all languages
	s.GenerateHreflang(appStore)
}

func addPage(now time.Time, pages *smg.Sitemap, s string, i float32, c smg.ChangeFreq) {
	err := pages.Add(&smg.SitemapLoc{
		Loc:        s,
		LastMod:    &now,
		ChangeFreq: c,
		Priority:   i,
	})
	if err != nil {
		slog.Error("Unable to add SitemapLoc:", "error", err)
	}
}
