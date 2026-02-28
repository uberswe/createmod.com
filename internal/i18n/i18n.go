package i18n

// Package i18n provides a minimal key-based translation lookup for UI strings.
// It relies on existing language detection (pages.preferredLanguageFromRequest)
// and exposes a simple T(lang, key) function for templates and handlers.

var translations = map[string]map[string]string{}

func init() {
	register("en", LangEN)
	register("pt-BR", LangPtBR)
	register("pt-PT", LangPtPT)
	register("es", LangES)
	register("de", LangDE)
	register("pl", LangPL)
	register("ru", LangRU)
	register("zh-Hans", LangZhHans)
}

func register(lang string, m map[string]string) {
	translations[lang] = m
}

// T returns the localized value for the given key in the provided lang.
// If no translation is found, it falls back to English, then to the key itself.
func T(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	if m, ok := translations["en"]; ok {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	return key
}
