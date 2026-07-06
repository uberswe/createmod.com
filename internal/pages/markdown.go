package pages

import (
	"fmt"
	"net/http"
	"strings"

	"createmod/internal/i18n"
	"createmod/internal/models"
	"createmod/internal/server"

	strip "github.com/grokify/html-strip-tags-go"
)

// wantsMarkdown reports whether the request negotiates a markdown response
// (Accept: text/markdown), as used by AI agents that prefer token-efficient
// representations over full HTML.
func wantsMarkdown(e *server.RequestEvent) bool {
	return strings.Contains(e.Request.Header.Get("Accept"), "text/markdown")
}

// serveMarkdown writes a markdown response with the negotiated content type
// and an estimated token count header (~4 chars per token).
func serveMarkdown(e *server.RequestEvent, md string) error {
	e.Response.Header().Set("X-Markdown-Tokens", fmt.Sprintf("%d", len(md)/4))
	e.Response.Header().Set("Vary", "Accept")
	return e.Blob(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}

// schematicMarkdown renders a curated markdown representation of a schematic
// page for agents: title, description, ratings, stats, video, mods and the
// material list. It deliberately excludes the schematic structure, block
// positions, and any NBT file or download URLs — agents get information about
// builds, never the builds themselves.
func schematicMarkdown(d SchematicData) string {
	s := d.Schematic
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", s.Title)
	fmt.Fprintf(&b, "A Minecraft Create Mod schematic on CreateMod.com.\n\n")
	fmt.Fprintf(&b, "- URL: https://createmod.com%s\n", PrefixedPath(d.Language, "/schematics/"+s.Name))
	if s.Author != nil && s.Author.Username != "" {
		fmt.Fprintf(&b, "- Author: %s (https://createmod.com/author/%s)\n", s.Author.Username, strings.ToLower(s.Author.Username))
	}
	if s.CreatedFormatted != "" {
		fmt.Fprintf(&b, "- Published: %s\n", s.CreatedFormatted)
	}
	if len(s.Categories) > 0 {
		names := make([]string, 0, len(s.Categories))
		for _, c := range s.Categories {
			names = append(names, c.Name)
		}
		fmt.Fprintf(&b, "- Categories: %s\n", strings.Join(names, ", "))
	}
	if len(s.Tags) > 0 {
		names := make([]string, 0, len(s.Tags))
		for _, t := range s.Tags {
			names = append(names, t.Name)
		}
		fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(names, ", "))
	}
	if s.HasRating && s.RatingCount > 0 {
		fmt.Fprintf(&b, "- Rating: %s/5 (%d ratings)\n", s.Rating, s.RatingCount)
	}
	fmt.Fprintf(&b, "- Views: %d\n", s.Views)
	if s.Downloads > 0 {
		fmt.Fprintf(&b, "- Downloads: %d\n", s.Downloads)
	}
	if s.MinecraftVersion != "" {
		fmt.Fprintf(&b, "- Minecraft version: %s\n", s.MinecraftVersion)
	}
	if s.CreatemodVersion != "" {
		fmt.Fprintf(&b, "- Create Mod version: %s\n", s.CreatemodVersion)
	}
	if s.BlockCount > 0 {
		fmt.Fprintf(&b, "- Block count: %d\n", s.BlockCount)
	}
	if s.DimX > 0 && s.DimY > 0 && s.DimZ > 0 {
		fmt.Fprintf(&b, "- Dimensions: %d x %d x %d\n", s.DimX, s.DimY, s.DimZ)
	}
	if len(s.Mods) > 0 {
		fmt.Fprintf(&b, "- Required mods: %s\n", strings.Join(s.Mods, ", "))
	}
	if vid := youtubeID(s.Video); vid != "" {
		fmt.Fprintf(&b, "- Video: https://www.youtube.com/watch?v=%s\n", vid)
	}

	if desc := strings.TrimSpace(strip.StripTags(s.Content)); desc != "" {
		fmt.Fprintf(&b, "\n## Description\n\n%s\n", desc)
	}

	if len(d.Materials) > 0 {
		b.WriteString("\n## Material list\n\n")
		for _, m := range d.Materials {
			fmt.Fprintf(&b, "- %s: %d\n", m.BlockID, m.Count)
		}
	}

	b.WriteString("\nThe schematic file itself (NBT structure and block placement data) is available to players through the website but is not provided to automated agents.\n")
	return b.String()
}

// indexMarkdown renders a curated markdown summary of the home page:
// site description plus the trending / latest / highest rated rails as
// linked schematic titles.
func indexMarkdown(d IndexData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# CreateMod.com - Minecraft Create Mod Schematics\n\n")
	fmt.Fprintf(&b, "%s\n", i18n.T(d.Language, "page.index.description"))

	appendRail := func(title string, schematics []models.Schematic) {
		if len(schematics) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n## %s\n\n", title)
		for i, s := range schematics {
			if i >= 12 {
				break
			}
			fmt.Fprintf(&b, "- [%s](https://createmod.com%s)", s.Title, PrefixedPath(d.Language, "/schematics/"+s.Name))
			if s.HasRating && s.RatingCount > 0 {
				fmt.Fprintf(&b, " — %s/5", s.Rating)
			}
			b.WriteString("\n")
		}
	}
	appendRail(i18n.T(d.Language, "Trending"), d.Trending)
	appendRail(i18n.T(d.Language, "Latest"), d.Schematics)
	appendRail(i18n.T(d.Language, "Highest Rated"), d.HighestRated)

	b.WriteString("\nAPI catalog: https://createmod.com/.well-known/api-catalog\n")
	return b.String()
}
