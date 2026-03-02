package pages

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// LangToPrefix maps internal language codes to URL path prefixes.
// English maps to "" (root), other languages to their subdirectory prefix.
var LangToPrefix = map[string]string{
	"en":      "",
	"de":      "de",
	"es":      "es",
	"pl":      "pl",
	"pt-BR":   "pt-br",
	"pt-PT":   "pt-pt",
	"ru":      "ru",
	"zh-Hans": "zh",
}

// PrefixToLang maps URL path prefixes back to internal language codes.
var PrefixToLang = map[string]string{
	"de":    "de",
	"es":    "es",
	"pl":    "pl",
	"pt-br": "pt-BR",
	"pt-pt": "pt-PT",
	"ru":    "ru",
	"zh":    "zh-Hans",
}

// HreflangEntry represents one <link rel="alternate" hreflang="..."> entry.
type HreflangEntry struct {
	HreflangCode string // e.g. "de", "pt-br", "zh-Hans"
	Prefix       string // e.g. "de", "pt-br", "zh", or "" for English
	Lang         string // internal language code e.g. "de", "pt-BR"
}

// FullPath returns the full path for this language given a bare path (no prefix).
func (h HreflangEntry) FullPath(barePath string) string {
	return PrefixedPath(h.Lang, barePath)
}

// allHreflangEntries is the static list of all supported hreflang entries.
var allHreflangEntries = []HreflangEntry{
	{HreflangCode: "en", Prefix: "", Lang: "en"},
	{HreflangCode: "de", Prefix: "de", Lang: "de"},
	{HreflangCode: "es", Prefix: "es", Lang: "es"},
	{HreflangCode: "pl", Prefix: "pl", Lang: "pl"},
	{HreflangCode: "pt-BR", Prefix: "pt-br", Lang: "pt-BR"},
	{HreflangCode: "pt-PT", Prefix: "pt-pt", Lang: "pt-PT"},
	{HreflangCode: "ru", Prefix: "ru", Lang: "ru"},
	{HreflangCode: "zh-Hans", Prefix: "zh", Lang: "zh-Hans"},
}

// AllHreflangs returns entries for all supported languages.
func AllHreflangs() []HreflangEntry {
	return allHreflangEntries
}

// PrefixedPath returns "/{prefix}{path}" for non-English, or path unchanged for English.
// The path should start with "/".
func PrefixedPath(lang, path string) string {
	prefix, ok := LangToPrefix[lang]
	if !ok || prefix == "" {
		return path
	}
	// Ensure path starts with /
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	if path == "/" {
		return "/" + prefix
	}
	return "/" + prefix + path
}

// StripLangPrefix extracts the language and remaining path from a URL path.
// Returns the detected language code and the path without the prefix.
// If no prefix matches, returns "" and the original path.
func StripLangPrefix(urlPath string) (lang string, stripped string) {
	if urlPath == "" || urlPath[0] != '/' {
		return "", urlPath
	}

	// Remove leading slash for matching
	rest := urlPath[1:]

	for prefix, langCode := range PrefixToLang {
		if rest == prefix {
			// Exact match: e.g. "/de"
			return langCode, "/"
		}
		if strings.HasPrefix(rest, prefix+"/") {
			// Prefix match: e.g. "/de/schematics"
			return langCode, "/" + rest[len(prefix)+1:]
		}
	}

	return "", urlPath
}

// LangRedirectURL builds a language-prefixed redirect target using the
// language detected from the request context (X-Createmod-Lang header).
func LangRedirectURL(e *core.RequestEvent, path string) string {
	lang := e.Request.Header.Get("X-Createmod-Lang")
	if lang == "" || !isSupportedLanguage(lang) {
		return path
	}
	return PrefixedPath(lang, path)
}
