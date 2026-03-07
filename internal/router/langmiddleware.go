package router

import (
	"createmod/internal/pages"
	"net/http"
	"strings"
)

// LangPrefixHandler wraps an http.Handler to strip language prefixes from
// the URL path before the request reaches the router's route matching.
// This must run at the HTTP transport level (before ServeMux) so that
// route matching sees the bare path (e.g. /schematics instead of /de/schematics).
func LangPrefixHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip paths that should never have language prefixes
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/_/") ||
			strings.HasPrefix(path, "/assets/") ||
			strings.HasPrefix(path, "/sitemaps/") ||
			path == "/robots.txt" ||
			path == "/ads.txt" {
			next.ServeHTTP(w, r)
			return
		}

		lang, stripped := pages.StripLangPrefix(path)
		if lang == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Redirect bare prefix (e.g. /de) to trailing-slash form (/de/)
		prefix := pages.LangToPrefix[lang]
		if path == "/"+prefix {
			target := "/" + prefix + "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}

		// Set language header for downstream handlers (Populate reads this)
		r.Header.Set("X-Createmod-Lang", lang)

		// Rewrite the URL path so ServeMux matches the bare route
		r.URL.Path = stripped
		r.URL.RawPath = ""

		next.ServeHTTP(w, r)
	})
}
