package router

import (
	"context"
	"createmod/internal/blockedurls"
	"createmod/internal/cache"
	"createmod/internal/pages"
	"createmod/internal/store"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// blockedURLMiddleware serves a 404 for requests whose path + query match an
// admin-managed blocked URL (e.g. DMCA takedown targets). It runs before the
// legacy-compat rewrites so the exact reported URL is blocked, not its
// redirect target. The blocklist is cached per-pod; admin changes invalidate
// the cache on all pods via Redis pub/sub.
func blockedURLMiddleware(appStore *store.Store, cacheService *cache.Service, notFound http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			entries := blockedURLList(appStore, cacheService)
			if len(entries) > 0 && blockedurls.Matches(entries, effectiveRequestURL(req)) {
				notFound(w, req)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// effectiveRequestURL returns the URL as the visitor requested it.
// LangPrefixHandler strips the language prefix (e.g. /es) at the transport
// level before this middleware runs, so blocked entries stored with a
// language prefix would never match without restoring it here.
func effectiveRequestURL(req *http.Request) *url.URL {
	lang := req.Header.Get("X-Createmod-Lang")
	if lang == "" {
		return req.URL
	}
	prefix, ok := pages.LangToPrefix[lang]
	if !ok || prefix == "" {
		return req.URL
	}
	u := *req.URL
	u.Path = "/" + prefix + u.Path
	return &u
}

// blockedURLList returns the blocked URL entries, from cache when possible.
// On store errors it fails open (returns nil) so a database hiccup never
// takes down regular traffic.
func blockedURLList(appStore *store.Store, cacheService *cache.Service) []string {
	if cacheService != nil {
		if v, ok := cacheService.Get(cache.BlockedURLsKey); ok {
			if entries, ok := v.([]string); ok {
				return entries
			}
		}
	}
	if appStore == nil || appStore.BlockedURLs == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entries, err := appStore.BlockedURLs.ListURLs(ctx)
	if err != nil {
		slog.Error("blocked urls: failed to list", "error", err)
		return nil
	}
	if entries == nil {
		entries = []string{}
	}
	if cacheService != nil {
		cacheService.Set(cache.BlockedURLsKey, entries)
	}
	return entries
}
