package sitemap

import (
	"context"
	"createmod/internal/storage"
	"createmod/internal/store"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sabloger/sitemap-generator/smg"
)

type Service struct {
	dev     bool
	storage *storage.Service
}

func New(dev bool, storageSvc *storage.Service) *Service {
	return &Service{dev: dev, storage: storageSvc}
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
	searches, err := appStore.SearchTracking.ListTopSearches(ctx, 10000)
	if err != nil {
		slog.Warn("sitemap: failed to list searches", "error", err)
	}
	guides, err := appStore.Guides.ListForSitemap(ctx)
	if err != nil {
		slog.Warn("sitemap: failed to list guides", "error", err)
	}
	collections, err := appStore.Collections.ListForSitemap(ctx)
	if err != nil {
		slog.Warn("sitemap: failed to list collections", "error", err)
	}
	now := time.Now().UTC()

	// Use a temp directory for the smg library to write to, then upload to S3
	tmpDir, err := os.MkdirTemp("", "sitemaps-*")
	if err != nil {
		slog.Error("sitemap: failed to create temp dir", "error", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	smi := smg.NewSitemapIndex(true)
	smi.SetCompress(false)
	smi.SetSitemapIndexName("sitemap")
	smi.SetHostname("https://www.createmod.com")
	smi.SetOutputPath(tmpDir + "/")
	smi.SetServerURI("/sitemaps/")

	smPages := smi.NewSitemap()
	smPages.SetName("pages")
	smPages.SetLastMod(&now)
	smPages.SetOutputPath(tmpDir + "/")

	addPage(now, smPages, "/", 1.0, smg.Daily)
	addPage(now, smPages, "/upload", 0.9, smg.Weekly)
	addPage(now, smPages, "/contact", 0.9, smg.Weekly)
	addPage(now, smPages, "/guides", 0.9, smg.Weekly)
	addPage(now, smPages, "/collections", 0.9, smg.Weekly)
	addPage(now, smPages, "/mods", 0.9, smg.Weekly)
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
	smSchematics.SetOutputPath(tmpDir + "/")
	smSchematics.SetLastMod(&now)

	for i := range schematics {
		if i != 0 && i%1000 == 0 {
			schematicsSmCount++
			smSchematics = smi.NewSitemap()
			smSchematics.SetName(fmt.Sprintf("schematics-%d", schematicsSmCount))
			smSchematics.SetOutputPath(tmpDir + "/")
			smSchematics.SetLastMod(&now)
		}
		lastMod := schematics[i].Updated
		err := smSchematics.Add(&smg.SitemapLoc{
			Loc:        fmt.Sprintf("/schematics/%s", schematics[i].Name),
			LastMod:    &lastMod,
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
	smUsers.SetOutputPath(tmpDir + "/")
	for i := range users {
		if i != 0 && i%1000 == 0 {
			userSmCount++
			smUsers = smi.NewSitemap()
			smUsers.SetName(fmt.Sprintf("users-%d", userSmCount))
			smUsers.SetLastMod(&now)
			smUsers.SetOutputPath(tmpDir + "/")
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
	smSearches.SetOutputPath(tmpDir + "/")

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

	// Guides sitemap
	if len(guides) > 0 {
		smGuides := smi.NewSitemap()
		smGuides.SetName("guides")
		smGuides.SetLastMod(&now)
		smGuides.SetOutputPath(tmpDir + "/")
		for _, g := range guides {
			lastMod := g.Updated
			if err := smGuides.Add(&smg.SitemapLoc{
				Loc:        fmt.Sprintf("/guides/%s", g.Slug),
				LastMod:    &lastMod,
				ChangeFreq: smg.Weekly,
				Priority:   0.7,
			}); err != nil {
				slog.Error("Unable to add guide SitemapLoc:", "error", err)
			}
		}
	}

	// Collections sitemap
	if len(collections) > 0 {
		smCollections := smi.NewSitemap()
		smCollections.SetName("collections")
		smCollections.SetLastMod(&now)
		smCollections.SetOutputPath(tmpDir + "/")
		for _, c := range collections {
			lastMod := c.Updated
			if err := smCollections.Add(&smg.SitemapLoc{
				Loc:        fmt.Sprintf("/collections/%s", c.Slug),
				LastMod:    &lastMod,
				ChangeFreq: smg.Weekly,
				Priority:   0.6,
			}); err != nil {
				slog.Error("Unable to add collection SitemapLoc:", "error", err)
			}
		}
	}

	// News sitemap
	smNews := smi.NewSitemap()
	smNews.SetName("news")
	smNews.SetLastMod(&now)
	smNews.SetOutputPath(tmpDir + "/")
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

	// Upload all generated files to S3
	s.uploadSitemapDir(ctx, tmpDir)

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

// uploadSitemapDir reads all files from the temp directory and uploads them to S3.
func (s *Service) uploadSitemapDir(ctx context.Context, dir string) {
	if s.storage == nil {
		slog.Warn("sitemap: no storage service, skipping S3 upload")
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Error("sitemap: failed to read temp dir", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			slog.Error("sitemap: failed to read file", "file", entry.Name(), "error", err)
			continue
		}
		key := "_sitemaps/" + entry.Name()
		if err := s.storage.UploadRawBytes(ctx, key, data, "application/xml"); err != nil {
			slog.Error("sitemap: failed to upload to S3", "file", entry.Name(), "error", err)
		} else {
			slog.Info("sitemap: uploaded to S3", "key", key)
		}
	}
}
