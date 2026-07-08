package pages

import (
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/models"
)

func Test_TruncateMetaDescription(t *testing.T) {
	if got := truncateMetaDescription("short description"); got != "short description" {
		t.Errorf("short strings must pass through, got %q", got)
	}
	long := strings.Repeat("word ", 60)
	got := truncateMetaDescription(long)
	if len(got) > metaDescriptionMaxLen+3 { // +3 for the ellipsis rune
		t.Errorf("expected truncation to ~%d chars, got %d", metaDescriptionMaxLen, len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated description should end with ellipsis, got %q", got)
	}
	// Whitespace (newlines, doubled spaces) is normalized
	if got := truncateMetaDescription("a\n\nb  c"); got != "a b c" {
		t.Errorf("expected whitespace normalization, got %q", got)
	}
}

func Test_SchematicJSONLD(t *testing.T) {
	d := SchematicData{}
	d.Language = "en"
	d.Description = "A great machine"
	d.Thumbnail = "https://createmod.com/api/files/schematics/x/img.webp"
	d.Schematic = models.Schematic{
		Title:       "Super Farm",
		Name:        "super-farm",
		Rating:      "4.5",
		RatingCount: 12,
		HasRating:   true,
		Video:       "https://www.youtube.com/watch?v=abc123DEF45",
		Author:      &models.User{Username: "alice"},
	}

	// aggregateRating is only valid under Google's supported review-snippet
	// parent types. Plain CreativeWork is not one (Search Console "Invalid
	// object type" error); MediaObject is, and being a CreativeWork subtype
	// it keeps author/video/dateCreated valid.
	out := string(d.SchematicJSONLD())
	for _, want := range []string{
		`"@type":"MediaObject"`,
		`"name":"Super Farm"`,
		`"aggregateRating"`,
		`"ratingValue":4.5`,
		`"ratingCount":12`,
		`"@type":"VideoObject"`,
		`"embedUrl":"https://www.youtube.com/embed/abc123DEF45"`,
		`"author"`,
		`"url":"https://createmod.com/author/alice"`,
		`"url":"https://createmod.com/schematics/super-farm"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON-LD missing %s in %s", want, out)
		}
	}
	for _, banned := range []string{`"@type":"CreativeWork"`, `"@type":"Product"`} {
		if strings.Contains(out, banned) {
			t.Errorf("rating parent must be MediaObject, found %s", banned)
		}
	}

	// Without rating or video, those blocks must be absent
	d.Schematic.HasRating = false
	d.Schematic.RatingCount = 0
	d.Schematic.Video = ""
	out = string(d.SchematicJSONLD())
	if strings.Contains(out, "aggregateRating") || strings.Contains(out, "VideoObject") {
		t.Errorf("expected no rating/video markup, got %s", out)
	}
	if !strings.Contains(out, `"@type":"MediaObject"`) || !strings.Contains(out, `"author"`) {
		t.Errorf("unrated JSON-LD should still be a MediaObject with author, got %s", out)
	}

	// Out-of-range averages (legacy 0-star rows made sub-1.0 averages
	// possible) must never emit a rating: Google requires
	// worstRating <= ratingValue <= bestRating (Search Console
	// "Rating value is out of range").
	for _, bad := range []string{"0.5", "0.0", "5.1", "12.0", "not-a-number"} {
		d.Schematic.HasRating = true
		d.Schematic.RatingCount = 3
		d.Schematic.Rating = bad
		out = string(d.SchematicJSONLD())
		if strings.Contains(out, "aggregateRating") {
			t.Errorf("rating %q must not emit aggregateRating", bad)
		}
	}
	// Boundary values are valid
	for _, good := range []string{"1.0", "5.0"} {
		d.Schematic.Rating = good
		out = string(d.SchematicJSONLD())
		if !strings.Contains(out, "aggregateRating") {
			t.Errorf("rating %q should emit aggregateRating", good)
		}
	}

	// French pages self-reference the prefixed URL
	d.Language = "fr"
	out = string(d.SchematicJSONLD())
	if !strings.Contains(out, `"url":"https://createmod.com/fr/schematics/super-farm"`) {
		t.Errorf("expected language-prefixed URL in JSON-LD, got %s", out)
	}
}

func Test_Head_Canonical_Is_Language_Aware(t *testing.T) {
	r := NewTestRegistry()
	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range rulesTemplates {
		paths = append(paths, filepath.Join(root, f))
	}

	d := RulesData{}
	d.Language = "fr"
	d.LangPrefix = "fr"
	d.Title = "Page Not Found"
	d.Slug = "/rules"

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(out, `<link rel="canonical" href="https://createmod.com/fr/rules">`) {
		t.Errorf("expected French page to self-canonicalize to /fr/rules")
	}
	if !strings.Contains(out, `<meta property="og:url" content="https://createmod.com/fr/rules">`) {
		t.Errorf("expected og:url to match canonical")
	}
	// Keyword-first title with brand suffix
	if !strings.Contains(out, "<title>Page Not Found - CreateMod.com</title>") {
		t.Errorf("expected keyword-first title format")
	}
	// No thumbnail set: og:image falls back to the site logo
	if !strings.Contains(out, `<meta property="og:image" content="https://createmod.com/assets/x/logo_sq_lg.png">`) {
		t.Errorf("expected og:image fallback to site logo")
	}
}
