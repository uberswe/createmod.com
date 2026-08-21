package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"createmod/internal/ratelimit"
)

// botFlagTTL is how long an IP stays flagged after it last reported a bot
// resolution. Short enough that a re-assigned residential IP frees up for a
// future real user; long enough to blunt a sustained flood.
const botFlagTTL = 12 * time.Hour

type botFlagCtxKeyT struct{}

var botFlagCtxKey = botFlagCtxKeyT{}

// WithBotFlag marks the request context as flagged-bot traffic (soft mode).
func WithBotFlag(ctx context.Context) context.Context {
	return context.WithValue(ctx, botFlagCtxKey, true)
}

// IsFlaggedBot reports whether this request was flagged as a bot. In soft mode
// the page is still served, but ads, analytics and view counting are skipped.
func IsFlaggedBot(ctx context.Context) bool {
	v, _ := ctx.Value(botFlagCtxKey).(bool)
	return v
}

// BotSignals are the no-permission client fingerprint fields the /api/ads-check
// beacon reports. Every one is readable without any browser permission prompt.
type BotSignals struct {
	Res   string `json:"res"`   // screen WxH
	WD    bool   `json:"wd"`    // navigator.webdriver — true under automation
	Langs int    `json:"langs"` // navigator.languages length
	HC    int    `json:"hc"`    // hardwareConcurrency
	DM    int    `json:"dm"`    // deviceMemory
	TP    int    `json:"tp"`    // maxTouchPoints
	GL    string `json:"gl"`    // WebGL UNMASKED_RENDERER
	TZ    string `json:"tz"`    // timezone
	Path  string `json:"path"`
}

// strongBotSignals reports near-certain headless/automation tells. These flag
// the IP (driving the download 403 + view-count skip), so the bar is high:
// navigator.webdriver, a software WebGL renderer (headless Chrome falls back to
// SwiftShader/llvmpipe), or the 800x600/square screen fingerprint.
func strongBotSignals(s BotSignals) bool {
	if s.WD {
		return true
	}
	gl := strings.ToLower(s.GL)
	if strings.Contains(gl, "swiftshader") || strings.Contains(gl, "llvmpipe") || strings.Contains(gl, "mesa offscreen") {
		return true
	}
	return isBotResolution(sanitizeResolution(s.Res))
}

// suppressAds is broader than strongBotSignals — it also treats absent language
// preferences as bot-like. Used ONLY for the fail-open ads/analytics decision,
// never to flag or block, so a false positive costs at most one missed ad
// impression, never access.
func suppressAds(s BotSignals) bool {
	return strongBotSignals(s) || s.Langs == 0
}

// botFlagKey is the (salted, hashed) Redis key for an IP — raw IPs are never
// stored.
func botFlagKey(ip string) string {
	sum := sha256.Sum256([]byte(securitySecret() + "|botflag|" + ip))
	return "botflag:" + hex.EncodeToString(sum[:16])
}

// isBotResolution reports whether a beacon resolution is a headless/bot
// fingerprint: the Puppeteer/headless-Chrome default 800x600, or any square
// WxH (no real display is square).
func isBotResolution(res string) bool {
	if res == "800x600" {
		return true
	}
	if w, h, ok := strings.Cut(res, "x"); ok && w != "" && w == h {
		return true
	}
	return false
}

// FlagBotIP records ip as a bot for botFlagTTL. Called from the pageview beacon
// when the client reports a bot resolution.
func FlagBotIP(ctx context.Context, rl ratelimit.Limiter, ip string) {
	if rl == nil || ip == "" {
		return
	}
	rl.Mark(ctx, botFlagKey(ip), botFlagTTL)
}

// IsBotFlaggedIP reports whether ip is currently flagged. Fails open (false) so
// a Redis outage never blocks or de-monetizes real users.
func IsBotFlaggedIP(ctx context.Context, rl ratelimit.Limiter, ip string) bool {
	if rl == nil || ip == "" {
		return false
	}
	return rl.Check(ctx, botFlagKey(ip))
}

// IsSchematicDownloadPath reports whether path serves an actual schematic file
// (not a page). Flagged bots get 403 on these; pages stay soft-served.
func IsSchematicDownloadPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/download/"):
		return true
	case strings.HasPrefix(path, "/api/download/"):
		return true
	case strings.HasPrefix(path, "/api/schematics/") && strings.HasSuffix(path, "/download"):
		return true
	case strings.HasPrefix(path, "/api/files/") && strings.HasSuffix(path, ".nbt"):
		return true
	default:
		return false
	}
}
