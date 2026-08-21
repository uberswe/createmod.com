package pages

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/traffic"
)

// trafficLangPrefixes are the URL language prefixes stripped before page
// classification (matches the site's supported locales).
var trafficLangPrefixes = map[string]bool{
	"en": true, "es": true, "fr": true, "pl": true, "ru": true,
	"pt-br": true, "pt-pt": true, "zh-hans": true,
}

// PageClass maps a request path to a coarse page group for traffic analysis.
// Low-cardinality on purpose — it keeps the traffic_stats key bounded.
func PageClass(path string) string {
	p := strings.Trim(path, "/")
	if p == "" {
		return "home"
	}
	seg := strings.SplitN(p, "/", 2)
	if trafficLangPrefixes[strings.ToLower(seg[0])] { // strip /es/, /pt-BR/, ...
		if len(seg) < 2 {
			return "home"
		}
		p = seg[1]
		seg = strings.SplitN(p, "/", 2)
	}
	switch strings.ToLower(seg[0]) {
	case "schematics":
		if len(seg) == 2 && seg[1] != "" {
			return "schematic"
		}
		return "browse"
	case "search", "tags", "tag", "categories", "category":
		return "browse"
	case "generators":
		return "generator"
	case "tools", "convert":
		return "tools"
	case "mods", "mod":
		return "mods"
	case "guides", "guide":
		return "guides"
	case "collections", "collection":
		return "collections"
	case "author", "settings", "user", "profile":
		return "user"
	default:
		return "other"
	}
}

var resolutionRe = regexp.MustCompile(`^\d{1,5}x\d{1,5}$`)

// sanitizeResolution returns 'WxH' only when it matches the expected shape, so
// a beacon can't poison the table with arbitrary text. Empty otherwise.
func sanitizeResolution(res string) string {
	if resolutionRe.MatchString(res) {
		return res
	}
	return ""
}

// AdsCheckHandler backs POST /api/ads-check. The client sends a no-permission
// fingerprint on page load; the server (1) records observability, (2) flags the
// IP when the signals are near-certain bot tells — which drives the server-side
// download 403 + view-count skip — and (3) tells the client whether to load ads
// and analytics. The client fails open (loads them on timeout/error), so the
// HTML stays identical/cacheable and real users never lose ads.
func AdsCheckHandler(rl ratelimit.Limiter) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var sig BotSignals
		_ = json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, 1024)).Decode(&sig)
		res := sanitizeResolution(sig.Res)
		ua, country, pageClass := e.Request.UserAgent(), e.Country(), PageClass(sig.Path)

		traffic.Record("view_js", ua, country, res, pageClass)
		if flagWorthy(sig) {
			FlagBotIP(e.Request.Context(), rl, e.RealIP())
			traffic.Record("bot_flag", ua, country, res, pageClass)
		}
		// Ad/search crawlers and ad-verification bots must always get ads —
		// suppressing them breaks AdSense targeting and IAS/DoubleVerify scores.
		serve := IsTrustedCrawler(ua) ||
			(!suppressAds(sig) && !IsBotFlaggedIP(e.Request.Context(), rl, e.RealIP()))
		return e.JSON(http.StatusOK, map[string]bool{"ads": serve})
	}
}
