package pages

import (
	"createmod/internal/i18n"
	html "html/template"
	"time"

	"github.com/pocketbase/pocketbase/tools/template"
)

// commonTemplates lists shared include fragments used by most pages.
var commonTemplates = []string{
	"./template/include/head.html",
	"./template/include/sidebar.html",
	"./template/include/header.html",
	"./template/include/footer.html",
	"./template/include/foot.html",
	"./template/include/ad_rail.html",
}

// NewTestRegistry creates a template registry with the FuncMap needed
// by common templates. Use in tests that render full pages.
func NewTestRegistry() *template.Registry {
	r := template.NewRegistry()
	r.AddFuncs(html.FuncMap{
		"AssetVer":  func() string { return "test" },
		"HumanDate": func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 MST") },
		"T":         func(lang, key string) string { return i18n.T(lang, key) },
		"SignedOutURL": func(rawURL string, args ...string) string {
			return "/out/test-token"
		},
		"tagSelected": func(selected []string, key string) bool {
			for _, s := range selected {
				if s == key {
					return true
				}
			}
			return false
		},
	})
	return r
}
