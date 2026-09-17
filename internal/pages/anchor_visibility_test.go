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

// Test_StickyFooter_IsSitewide guards the two settings that make the sticky
// footer actually appear everywhere. It originally ran only on mobile/tablet
// via a max-width mediaQuery, and the legacy "anchor" format created the bar
// but never filled it, so both are pinned here.
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

	// NitroPay's ad builder emits anchor-v2; the legacy "anchor" format does not
	// fill (it renders an empty bar).
	if !strings.Contains(cfg, `"format": "anchor-v2"`) {
		t.Errorf("cm-anchor must use the anchor-v2 format:\n%s", cfg)
	}
	if !strings.Contains(cfg, `"anchor": "bottom"`) {
		t.Errorf("cm-anchor is not a bottom anchor:\n%s", cfg)
	}
	// A max-width mediaQuery would cap the footer to mobile again. A min-width
	// of 0 is how the builder expresses "every width" and is expected.
	if strings.Contains(cfg, "max-width") {
		t.Errorf("cm-anchor has a max-width cap; the sticky footer must run at all widths:\n%s", cfg)
	}
}

// Test_NoTopBanners guards that the 728x90 top leaderboards stay removed
// sitewide: they were replaced by the sticky footer anchor. The editor's
// in-content horizontal unit (below the 3D canvas, not a top banner) is
// deliberately excluded — it is the editor's only desktop placement.
func Test_NoTopBanners(t *testing.T) {
	root := projectRootFromThisFile(t)
	pages, err := filepath.Glob(filepath.Join(root, "template", "*.html"))
	if err != nil {
		t.Fatalf("glob templates: %v", err)
	}
	for _, p := range pages {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		s := string(raw)
		name := filepath.Base(p)
		for _, marker := range []string{"-top-banner", "cm-top-banner"} {
			if strings.Contains(s, marker) {
				t.Errorf("%s still references %q; top leaderboards were replaced by the sticky footer", name, marker)
			}
		}
	}
}
