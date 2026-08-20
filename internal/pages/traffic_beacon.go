package pages

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

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

// PageviewBeaconHandler records the client-side pageview beacon
// (POST /api/pageview, body {res, path}). It captures screen resolution — which
// only JS-capable clients can report — as event_type "view_js" for
// bot-traffic analysis. Best-effort: always 204, never errors the client.
func PageviewBeaconHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var body struct {
			Res  string `json:"res"`
			Path string `json:"path"`
		}
		_ = json.NewDecoder(http.MaxBytesReader(e.Response, e.Request.Body, 512)).Decode(&body)
		traffic.Record("view_js", e.Request.UserAgent(), e.Country(), sanitizeResolution(body.Res), PageClass(body.Path))
		return e.NoContent(http.StatusNoContent)
	}
}
