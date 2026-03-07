package sitemap

import (
	"context"
	"createmod/internal/pages"
	"createmod/internal/store"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// hreflangURLSet is the root element for a sitemap with xhtml:link alternates.
type hreflangURLSet struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	XHTML   string        `xml:"xmlns:xhtml,attr"`
	URLs    []hreflangURL `xml:"url"`
}

type hreflangURL struct {
	Loc   string          `xml:"loc"`
	Links []hreflangLink  `xml:"xhtml:link"`
}

type hreflangLink struct {
	Rel      string `xml:"rel,attr"`
	Hreflang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

const sitemapOutputPath = "template/dist/sitemaps/"

// GenerateHreflang creates sitemap files with xhtml:link hreflang alternates
// for all supported languages. Each language gets its own sitemap file.
func (s *Service) GenerateHreflang(appStore *store.Store) {
	slog.Info("hreflang sitemap generation started")

	hostname := "https://createmod.com"
	entries := pages.AllHreflangs()

	// Collect all schematic paths
	ctx := context.Background()
	schematics, err := appStore.Schematics.ListForSitemap(ctx)
	if err != nil {
		slog.Warn("hreflang sitemap: failed to query schematics", "error", err)
	}

	// Static pages to include in hreflang sitemaps
	staticPaths := []string{
		"/",
		"/upload",
		"/contact",
		"/rules",
		"/terms-of-service",
		"/privacy-policy",
		"/login",
		"/register",
		"/reset-password",
		"/news",
		"/schematics",
		"/search",
		"/explore",
		"/users",
		"/videos",
		"/guides",
		"/collections",
		"/mods",
	}

	// Build all bare paths (without language prefix)
	var allPaths []string
	allPaths = append(allPaths, staticPaths...)
	for _, sc := range schematics {
		allPaths = append(allPaths, "/schematics/"+sc.Name)
	}

	// buildLinks creates the hreflang link set for a given bare path
	buildLinks := func(barePath string) []hreflangLink {
		links := make([]hreflangLink, 0, len(entries)+1)
		for _, entry := range entries {
			links = append(links, hreflangLink{
				Rel:      "alternate",
				Hreflang: entry.HreflangCode,
				Href:     hostname + pages.PrefixedPath(entry.Lang, barePath),
			})
		}
		// x-default points to English (root)
		links = append(links, hreflangLink{
			Rel:      "alternate",
			Hreflang: "x-default",
			Href:     hostname + barePath,
		})
		return links
	}

	// Generate one sitemap per language
	for _, entry := range entries {
		urlSet := hreflangURLSet{
			XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
			XHTML: "http://www.w3.org/1999/xhtml",
		}

		// Split into chunks of 5000 to stay within sitemap limits
		chunkIdx := 0
		for i, barePath := range allPaths {
			if i > 0 && i%5000 == 0 {
				// Write current chunk
				if err := writeHreflangSitemap(entry, chunkIdx, urlSet); err != nil {
					slog.Error("hreflang sitemap write failed", "lang", entry.Lang, "chunk", chunkIdx, "error", err)
				}
				chunkIdx++
				urlSet.URLs = nil
			}

			loc := hostname + pages.PrefixedPath(entry.Lang, barePath)
			urlSet.URLs = append(urlSet.URLs, hreflangURL{
				Loc:   loc,
				Links: buildLinks(barePath),
			})
		}

		// Write final chunk
		if len(urlSet.URLs) > 0 {
			if err := writeHreflangSitemap(entry, chunkIdx, urlSet); err != nil {
				slog.Error("hreflang sitemap write failed", "lang", entry.Lang, "chunk", chunkIdx, "error", err)
			}
		}
	}

	slog.Info("hreflang sitemap generation completed")
}

func writeHreflangSitemap(entry pages.HreflangEntry, chunkIdx int, urlSet hreflangURLSet) error {
	prefix := entry.Prefix
	if prefix == "" {
		prefix = "en"
	}

	var filename string
	if chunkIdx == 0 {
		filename = fmt.Sprintf("hreflang-%s.xml", prefix)
	} else {
		filename = fmt.Sprintf("hreflang-%s-%d.xml", prefix, chunkIdx+1)
	}

	path := filepath.Join(sitemapOutputPath, filename)

	// Ensure output directory exists
	if err := os.MkdirAll(sitemapOutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create sitemap directory: %w", err)
	}

	data, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hreflang sitemap: %w", err)
	}

	content := xml.Header + string(data)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write hreflang sitemap: %w", err)
	}

	slog.Info("hreflang sitemap written", "file", filename, "urls", len(urlSet.URLs))
	return nil
}
