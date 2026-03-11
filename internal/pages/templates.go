package pages

import (
	"createmod/internal/i18n"
	html "html/template"
	"net/url"
	"strings"
	"time"

	"createmod/internal/server"
)

// commonTemplates lists shared include fragments used by most pages.
var commonTemplates = []string{
	"./template/include/head.html",
	"./template/include/sidebar.html",
	"./template/include/header.html",
	"./template/include/footer.html",
	"./template/include/foot.html",
}

// NewTestRegistry creates a template registry with the FuncMap needed
// by common templates. Use in tests that render full pages.
func NewTestRegistry() *server.Registry {
	r := server.NewRegistry()
	r.AddFuncs(html.FuncMap{
		"AssetVer":  func() string { return "test" },
		"HumanDate": func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 MST") },
		"DateOnly":  func(t time.Time) string { return t.UTC().Format("2006-01-02") },
		"T":         func(lang, key string) string { return i18n.T(lang, key) },
		"ToLower":   strings.ToLower,
		"mod":       func(i, j int) bool { return i%j == 0 },
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
		"LangURL": func(lang string, path string) string {
			return PrefixedPath(lang, path)
		},
		"Hreflangs": func(barePath string) []HreflangEntry {
			return AllHreflangs()
		},
		"urlPathEscape":  url.PathEscape,
		"externalDomain": ExternalDomain,
		"LangFlag": func(code string) html.HTML {
			cc := "gb"
			switch code {
			case "en":
				cc = "gb"
			case "pt-BR":
				cc = "br"
			case "pt-PT":
				cc = "pt"
			case "es":
				cc = "es"
			case "de":
				cc = "de"
			case "pl":
				cc = "pl"
			case "ru":
				cc = "ru"
			case "zh-Hans":
				cc = "cn"
			}
			return html.HTML(`<span class="fi fi-` + cc + `"></span>`)
		},
	})
	return r
}

// ExternalDomain extracts a human-friendly platform name from a URL.
func ExternalDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")

	switch {
	case strings.Contains(host, "patreon.com"):
		return "Patreon"
	case strings.Contains(host, "ko-fi.com"):
		return "Ko-fi"
	case strings.Contains(host, "discord.gg"), strings.Contains(host, "discord.com"):
		return "Discord"
	case strings.Contains(host, "gumroad.com"):
		return "Gumroad"
	case strings.Contains(host, "buymeacoffee.com"):
		return "Buy Me a Coffee"
	default:
		return host
	}
}
