package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/server"
	"fmt"
	htmltmpl "html/template"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// uploadPendingFuncMap mirrors the subset of the production router funcmap
// (internal/router/main.go) that the pending page and its common templates use.
func uploadPendingFuncMap() htmltmpl.FuncMap {
	return htmltmpl.FuncMap{
		"ToLower":         strings.ToLower,
		"mod":             func(i, j int) bool { return i%j == 0 },
		"add":             func(a, b int) int { return a + b },
		"sub":             func(a, b int) int { return a - b },
		"FormatNumber":    func(n int) string { return fmt.Sprintf("%d", n) },
		"urlPathEscape":   func(s string) string { return s },
		"HumanDate":       func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 MST") },
		"DateOnly":        func(t time.Time) string { return t.UTC().Format("2006-01-02") },
		"NewsDate":        func(t time.Time) string { return t.UTC().Format("January 2, 2006") },
		"printf":          fmt.Sprintf,
		"T":               func(lang, key string) string { return i18n.T(lang, key) },
		"AssetVer":        func() string { return "test" },
		"SignedOutURL":    func(rawURL string, args ...string) string { return rawURL },
		"tagSelected":     func(selected []string, key string) bool { return false },
		"LangURL":         func(lang, path string) string { return PrefixedPath(lang, path) },
		"Hreflangs":       func(barePath string) []HreflangEntry { return AllHreflangs() },
		"YouTubeWatchURL": func(video string) string { return "" },
		"externalDomain":  func(u string) string { return "" },
		"PlaceholderImg":  func(id string) string { return "" },
		"LangFlag":        func(code string) htmltmpl.HTML { return "" },
		"LangName":        func(code string) string { return code },
	}
}

// Test_UploadPending_FullPageRenders guards against the pending page 500ing on
// a missing template or missing data field. It loads the exact template set the
// handler uses (uploadPendingTemplates) and renders a populated UploadPendingData,
// which is a superset execute of every {{ template }} include and every field
// reference in that tree. Regression for #1646: upload_steps.html was omitted
// from the load list, and UploadStep was missing from UploadPendingData.
func Test_UploadPending_FullPageRenders(t *testing.T) {
	root := projectRootFromThisFile(t)
	files := make([]string, 0, len(uploadPendingTemplates))
	for _, f := range uploadPendingTemplates {
		files = append(files, filepath.Join(root, strings.TrimPrefix(f, "./")))
	}

	r := server.NewRegistry()
	r.AddFuncs(uploadPendingFuncMap())

	d := UploadPendingData{
		UploadStep:       3,
		SchematicName:    "beginner-iron-farm",
		SchematicURL:     "/schematics/beginner-iron-farm",
		SchematicFullURL: "https://createmod.com/schematics/beginner-iron-farm",
		SchematicID:      "bc56846fc2ff489",
		Outcome:          "limited",
		HeroLevel:        "warn",
		HeroTitle:        "Published with limits",
		HeroBody:         "Visible via direct link only.",
		HeldImageCount:   0,
		SLAHours:         48,
		PollActive:       false,
	}
	d.Language = "en"

	if _, err := r.LoadFiles(files...).Render(d); err != nil {
		t.Fatalf("pending page render failed: %v", err)
	}
}
