package i18n

// Package i18n provides a minimal key-based translation lookup for UI strings.
// It relies on existing language detection (pages.preferredLanguageFromRequest)
// and exposes a simple T(lang, key) function for templates and handlers.

var translations = map[string]map[string]string{
	"en": {
		"Read more":       "Read more",
		"No news yet":     "No news yet",
		"Back to News":    "Back to News",
		"Copy Link":       "Copy Link",
		"Download":        "Download",
		"Get Schematic":   "Get Schematic",
		"Report":          "Report",
		"Edit":            "Edit",
		"Version history": "Version history",
	},
	"pt-BR": {
		"Read more":       "Ler mais",
		"No news yet":     "Ainda não há notícias",
		"Back to News":    "Voltar para Notícias",
		"Copy Link":       "Copiar link",
		"Download":        "Baixar",
		"Get Schematic":   "Obter Esquemático",
		"Report":          "Denunciar",
		"Edit":            "Editar",
		"Version history": "Histórico de versões",
	},
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
