package pages

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// footAnchorRendered renders include/foot.html against the given DefaultData
// flags and reports whether the sitewide sticky footer (cm-anchor) was emitted.
func footAnchorRendered(t *testing.T, d DefaultData) string {
	t.Helper()
	path := filepath.Join(projectRootFromThisFile(t), "template", "include", "foot.html")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read foot.html: %v", err)
	}
	tmpl, err := template.New("foot.html").
		Funcs(template.FuncMap{
			"T":        func(lang, key string) string { return key },
			"AssetVer": func() string { return "test" },
			"LangURL":  func(lang, path string) string { return path },
		}).
		Parse(string(raw))
	if err != nil {
		t.Fatalf("parse foot.html: %v", err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, d); err != nil {
		t.Fatalf("execute foot.html: %v", err)
	}
	return sb.String()
}

// Test_StickyFooter_Visibility pins the sticky-footer (bottom anchor) rules:
// it runs sitewide, but never on the upload forms, auth/settings flows
// (HideAnchor) or admin/settings pages (NoAds). It must also be independent of
// HideOutstream, which only suppresses the floating video player.
func Test_StickyFooter_Visibility(t *testing.T) {
	cases := []struct {
		name string
		data DefaultData
		want bool
	}{
		{"default page shows the sticky footer", DefaultData{}, true},
		{"video suppressed but anchor kept (e.g. homepage)", DefaultData{HideOutstream: true}, true},
		{"upload/auth flows hide the anchor", DefaultData{HideAnchor: true}, false},
		{"admin and settings pages hide the whole stack", DefaultData{NoAds: true, HideOutstream: true, HideAnchor: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := footAnchorRendered(t, c.data)
			got := strings.Contains(out, "createAd('cm-anchor'")
			if got != c.want {
				t.Errorf("sticky footer rendered = %v, want %v", got, c.want)
			}
		})
	}
}

// Test_StickyFooter_IsSitewide guards that the anchor carries no width cap: it
// previously ran only on mobile/tablet via a max-width mediaQuery.
func Test_StickyFooter_IsSitewide(t *testing.T) {
	out := footAnchorRendered(t, DefaultData{})
	start := strings.Index(out, "createAd('cm-anchor'")
	if start < 0 {
		t.Fatal("cm-anchor unit not found")
	}
	end := strings.Index(out[start:], "});")
	if end < 0 {
		t.Fatal("could not find end of cm-anchor config")
	}
	cfg := out[start : start+end]
	if strings.Contains(cfg, "mediaQuery") {
		t.Errorf("cm-anchor still has a mediaQuery cap; the sticky footer must run at all widths:\n%s", cfg)
	}
	if !strings.Contains(cfg, `"anchor": "bottom"`) {
		t.Errorf("cm-anchor is not a bottom anchor:\n%s", cfg)
	}
}

// Test_SchematicPage_NoTopBanner guards that the schematic page's 728x90 top
// leaderboard stays removed (replaced by the sticky footer).
func Test_SchematicPage_NoTopBanner(t *testing.T) {
	path := filepath.Join(projectRootFromThisFile(t), "template", "schematic.html")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schematic.html: %v", err)
	}
	s := string(raw)
	if strings.Contains(s, "schematic-top-banner") {
		t.Error("schematic.html still references schematic-top-banner; it was replaced by the sticky footer")
	}
}
