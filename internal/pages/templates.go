package pages

import (
	"createmod/internal/i18n"
	"crypto/sha256"
	"fmt"
	html "html/template"
	"net/url"
	"regexp"
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
	"./template/include/admin_nav.html",
	"./template/include/settings_nav.html",
	"./template/include/user_badges.html",
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
		"add":       func(a, b int) int { return a + b },
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
		"urlPathEscape": url.PathEscape,
		"YouTubeWatchURL": func(video string) string {
			return YoutubeWatchURL(video)
		},
		"externalDomain": ExternalDomain,
		"PlaceholderImg": func(id string) string {
			themes := [8]string{"brass", "cobble", "copper", "forest", "night", "redst", "sand", "sky"}
			h := sha256.Sum256([]byte(id))
			idx := int(h[0]) % 64
			theme := themes[idx/8]
			num := idx%8 + 1
			return fmt.Sprintf("/assets/x/placeholders/schematic-%s-%02d.svg", theme, num)
		},
		"LangFlag": func(code string) html.HTML {
			return ""
		},
		"LangName": func(code string) string {
			switch code {
			case "en":
				return "English"
			case "de":
				return "Deutsch"
			case "es":
				return "Español"
			case "fr":
				return "Français"
			case "pl":
				return "Polski"
			case "pt-BR":
				return "Português (Brasil)"
			case "pt-PT":
				return "Português (Portugal)"
			case "ru":
				return "Русский"
			case "zh-Hans":
				return "简体中文"
			default:
				return code
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

func ExtractYouTubeID(video string) string {
	video = strings.TrimSpace(video)
	if video == "" {
		return ""
	}
	if u, err := url.Parse(video); err == nil && u.Host != "" {
		host := strings.ToLower(u.Host)
		switch {
		case host == "youtu.be" || host == "www.youtu.be":
			id := strings.TrimPrefix(u.Path, "/")
			if id != "" {
				return id
			}
		case strings.Contains(host, "youtube.com"):
			if strings.HasPrefix(u.Path, "/embed/") {
				id := strings.TrimPrefix(u.Path, "/embed/")
				if id != "" {
					return id
				}
			}
			if strings.HasPrefix(u.Path, "/shorts/") {
				id := strings.TrimPrefix(u.Path, "/shorts/")
				if id != "" {
					return id
				}
			}
			if v := u.Query().Get("v"); v != "" {
				return v
			}
		}
	}
	return video
}

var youtubeIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func IsValidYouTubeVideo(video string) bool {
	video = strings.TrimSpace(video)
	if video == "" {
		return true
	}
	u, err := url.Parse(video)
	if err == nil && u.Host != "" {
		host := strings.ToLower(u.Host)
		switch {
		case host == "youtu.be" || host == "www.youtu.be":
			return strings.TrimPrefix(u.Path, "/") != ""
		case strings.Contains(host, "youtube.com"):
			if strings.HasPrefix(u.Path, "/embed/") {
				return strings.TrimPrefix(u.Path, "/embed/") != ""
			}
			if strings.HasPrefix(u.Path, "/shorts/") {
				return strings.TrimPrefix(u.Path, "/shorts/") != ""
			}
			return u.Query().Get("v") != ""
		}
		return false
	}
	return youtubeIDRegex.MatchString(video)
}

func YoutubeWatchURL(video string) string {
	id := ExtractYouTubeID(video)
	if id == "" {
		return video
	}
	return "https://www.youtube.com/watch?v=" + id
}
