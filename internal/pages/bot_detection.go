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

// botSignalCount tallies independent near-certain headless/automation tells:
// navigator.webdriver, a software WebGL renderer (headless Chrome falls back to
// SwiftShader/llvmpipe), and the 800x600/square screen fingerprint. Each is
// counted at most once.
func botSignalCount(s BotSignals) int {
	n := 0
	if s.WD {
		n++
	}
	gl := strings.ToLower(s.GL)
	if strings.Contains(gl, "swiftshader") || strings.Contains(gl, "llvmpipe") || strings.Contains(gl, "mesa offscreen") {
		n++
	}
	if isBotResolution(sanitizeResolution(s.Res)) {
		n++
	}
	return n
}

// flagWorthy reports whether the fingerprint is strong enough to FLAG the IP —
// which drives the download 403 and view-count skip. It requires >=2 independent
// tells on purpose. Any single signal can come from a real user (a browser with
// GPU acceleration off reports SwiftShader identically to headless Chrome) or
// from one bot sharing an IP with real users (CGNAT, mobile carrier, office,
// household) — flagging on it would then deny downloads to everyone behind that
// IP. Requiring two independent tells makes a genuine headless client
// overwhelmingly likely before we block anything.
func flagWorthy(s BotSignals) bool {
	return botSignalCount(s) >= 2
}

// suppressAds is the softer, fail-safe ads/analytics decision. A SINGLE tell (or
// absent language preferences) is enough, because a false positive here costs at
// most one missed ad impression, never access — so the bar is deliberately lower
// than flagWorthy.
func suppressAds(s BotSignals) bool {
	return botSignalCount(s) >= 1 || s.Langs == 0
}

// trustedCrawlerUAs are lowercased substrings of ad/search crawlers and
// ad-verification bots that MUST always be served ads: the AdSense crawler
// (Mediapartners-Google) renders via headless Chrome, so it would otherwise
// fingerprint as a bot and get a no-ads page — hurting ad targeting — and
// blocking IAS/DoubleVerify/Moat breaks ad-viewability verification scores.
var trustedCrawlerUAs = []string{
	"mediapartners-google", // AdSense content crawler (renders via headless Chrome)
	"adsbot-google",        // Google Ads landing-page crawler
	"googlebot",            // Google Search (also renders via headless Chrome)
	"google-inspectiontool",
	"apis-google",
	"bingbot",
	"adidxbot",
	// Ad-verification vendors.
	"ias_crawler", "ias-va", "ias-ie", "integralads", "admantx", // Integral Ad Science
	"doubleverify",
	"moatbot", "moat.com",
	"grapeshot",
	"proximic",
}

// IsTrustedCrawler reports whether ua is a known ad/search crawler that must
// always be served ads. Such clients may still be flagged (so a UA-spoofing bot
// can't borrow these names to evade the download 403), but the ads decision is
// forced on for them.
func IsTrustedCrawler(ua string) bool {
	l := strings.ToLower(ua)
	for _, s := range trustedCrawlerUAs {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
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
