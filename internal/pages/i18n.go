package pages

import (
	"net/http"
	"strings"
)

var supportedLanguages = map[string]struct{}{
	"en":      {},
	"fr":      {},
	"pt-BR":   {},
	"pt-PT":   {},
	"es":      {},
	"de":      {},
	"pl":      {},
	"ru":      {},
	"zh-Hans": {},
}

func isSupportedLanguage(code string) bool {
	_, ok := supportedLanguages[code]
	return ok
}

// normalizeFromAcceptLanguage maps common Accept-Language header values to our supported set.
func normalizeFromAcceptLanguage(header string) string {
	h := strings.TrimSpace(strings.ToLower(header))
	if h == "" {
		return "en"
	}
	// take first token before comma
	if idx := strings.Index(h, ","); idx >= 0 {
		h = h[:idx]
	}
	h = strings.TrimSpace(h)
	switch {
	case strings.HasPrefix(h, "pt-br"):
		return "pt-BR"
	case strings.HasPrefix(h, "pt-pt"):
		return "pt-PT"
	case h == "pt" || strings.HasPrefix(h, "pt-"):
		return "pt-PT"
	case strings.HasPrefix(h, "fr"):
		return "fr"
	case strings.HasPrefix(h, "es"):
		return "es"
	case strings.HasPrefix(h, "de"):
		return "de"
	case strings.HasPrefix(h, "pl"):
		return "pl"
	case strings.HasPrefix(h, "ru"):
		return "ru"
	case strings.HasPrefix(h, "zh"):
		return "zh-Hans"
	default:
		return "en"
	}
}

// preferredLanguageFromRequest returns the cookie value if present and supported, else "en".
// Language is only changed when the user explicitly selects one (stored in the cm_lang cookie).
func preferredLanguageFromRequest(r *http.Request) string {
	if r == nil {
		return "en"
	}
	if c, err := r.Cookie("cm_lang"); err == nil {
		v := strings.TrimSpace(c.Value)
		if isSupportedLanguage(v) {
			return v
		}
	}
	return "en"
}
