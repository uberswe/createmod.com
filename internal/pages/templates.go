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
	"./template/include/ad_rail.html",
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
		"externalDomain": ExternalDomain,
		"LangFlag": func(code string) string {
			switch code {
			case "en":
				return "\U0001F1EC\U0001F1E7"
			case "pt-BR":
				return "\U0001F1E7\U0001F1F7"
			case "pt-PT":
				return "\U0001F1F5\U0001F1F9"
			case "es":
				return "\U0001F1EA\U0001F1F8"
			case "de":
				return "\U0001F1E9\U0001F1EA"
			case "pl":
				return "\U0001F1F5\U0001F1F1"
			case "ru":
				return "\U0001F1F7\U0001F1FA"
			case "zh-Hans":
				return "\U0001F1E8\U0001F1F3"
			default:
				return "\U0001F310"
			}
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
