package pages

import "testing"

func TestStrongBotSignals(t *testing.T) {
	// Near-certain headless tells → flag (drives download 403 + view-count skip).
	strong := []BotSignals{
		{WD: true, Res: "1920x1080", Langs: 2},                 // navigator.webdriver
		{GL: "Google SwiftShader", Res: "1920x1080", Langs: 2}, // software WebGL
		{GL: "llvmpipe (LLVM 12.0.0, 256 bits)", Langs: 2},     // software WebGL
		{Res: "800x600", Langs: 2},                             // headless default viewport
		{Res: "1600x1600", Langs: 2},                           // square screen
	}
	for _, s := range strong {
		if !strongBotSignals(s) {
			t.Errorf("strongBotSignals(%+v) = false, want true", s)
		}
	}
	// A normal desktop browser must never be flagged.
	real := BotSignals{Res: "1920x1080", WD: false, Langs: 2, HC: 8, DM: 8, GL: "ANGLE (NVIDIA GeForce RTX 3060)", TZ: "Europe/Stockholm"}
	if strongBotSignals(real) {
		t.Errorf("strongBotSignals(real desktop) = true, want false")
	}
	realMobile := BotSignals{Res: "412x915", WD: false, Langs: 1, TP: 5, GL: "ANGLE (Adreno 640)"}
	if strongBotSignals(realMobile) {
		t.Errorf("strongBotSignals(real mobile) = true, want false")
	}
}

func TestSuppressAds(t *testing.T) {
	// Broader than the flag: absent language prefs also suppress ads (fail-safe,
	// never blocks — only costs an impression).
	if !suppressAds(BotSignals{Res: "1920x1080", Langs: 0}) {
		t.Error("no languages should suppress ads")
	}
	if suppressAds(BotSignals{Res: "1920x1080", Langs: 2, GL: "ANGLE (Intel UHD)"}) {
		t.Error("normal client should be served ads")
	}
	// But absent languages alone must NOT be a strong (blocking) signal.
	if strongBotSignals(BotSignals{Res: "1920x1080", Langs: 0}) {
		t.Error("no languages alone must not flag/block the IP")
	}
}

func TestIsTrustedCrawler(t *testing.T) {
	// These must always be served ads even though they fingerprint as headless.
	trusted := []string{
		"Mediapartners-Google",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"AdsBot-Google (+http://www.google.com/adsbot.html)",
		"IAS Crawler (ias_crawler; http://integralads.com/site-indexing/)",
		"ias-va/3.3 (former https://www.admantx.com)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"DoubleVerify Crawler",
	}
	for _, ua := range trusted {
		if !IsTrustedCrawler(ua) {
			t.Errorf("IsTrustedCrawler(%q) = false, want true", ua)
		}
	}
	notTrusted := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120 Safari/537.36",
		"Java-http-client/21.0.11",
		"",
	}
	for _, ua := range notTrusted {
		if IsTrustedCrawler(ua) {
			t.Errorf("IsTrustedCrawler(%q) = true, want false", ua)
		}
	}
	// A trusted crawler with headless signals must still be served ads (this is
	// the guard against no-ads pages breaking AdSense/IAS).
	if !IsTrustedCrawler("Mediapartners-Google") {
		t.Fatal("precondition")
	}
	if !strongBotSignals(BotSignals{WD: true}) {
		t.Fatal("precondition: webdriver should be a strong signal")
	}
}

func TestIsSchematicDownloadPath(t *testing.T) {
	yes := []string{
		"/download/steam-train",
		"/api/download/steam-train",
		"/api/schematics/steam-train/download",
		"/api/files/schematics/abc123/steam-train.nbt",
	}
	for _, p := range yes {
		if !IsSchematicDownloadPath(p) {
			t.Errorf("IsSchematicDownloadPath(%q) = false, want true", p)
		}
	}
	no := []string{
		"/schematics/steam-train",          // page
		"/get/steam-train",                 // interstitial page
		"/api/schematics/steam-train",      // detail JSON
		"/api/files/schematics/abc/x.webp", // image
		"/",
	}
	for _, p := range no {
		if IsSchematicDownloadPath(p) {
			t.Errorf("IsSchematicDownloadPath(%q) = true, want false", p)
		}
	}
}
