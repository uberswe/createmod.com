package pages

import (
	"testing"
)

func TestPrefixedPath(t *testing.T) {
	tests := []struct {
		lang string
		path string
		want string
	}{
		{"en", "/", "/"},
		{"en", "/schematics", "/schematics"},
		{"de", "/", "/de"},
		{"de", "/schematics", "/de/schematics"},
		{"de", "/schematics/my-build", "/de/schematics/my-build"},
		{"pt-BR", "/", "/pt-br"},
		{"pt-BR", "/search/test", "/pt-br/search/test"},
		{"pt-PT", "/upload", "/pt-pt/upload"},
		{"zh-Hans", "/guides", "/zh/guides"},
		{"ru", "/login", "/ru/login"},
		{"unknown", "/test", "/test"}, // unsupported language falls back
	}
	for _, tt := range tests {
		got := PrefixedPath(tt.lang, tt.path)
		if got != tt.want {
			t.Errorf("PrefixedPath(%q, %q) = %q, want %q", tt.lang, tt.path, got, tt.want)
		}
	}
}

func TestStripLangPrefix(t *testing.T) {
	tests := []struct {
		urlPath     string
		wantLang    string
		wantStripped string
	}{
		{"/de/schematics", "de", "/schematics"},
		{"/de", "de", "/"},
		{"/de/", "de", "/"},
		{"/pt-br/search/test", "pt-BR", "/search/test"},
		{"/pt-pt/upload", "pt-PT", "/upload"},
		{"/zh/guides", "zh-Hans", "/guides"},
		{"/ru/login", "ru", "/login"},
		{"/es/collections/my-stuff", "es", "/collections/my-stuff"},
		// No prefix
		{"/schematics", "", "/schematics"},
		{"/", "", "/"},
		{"", "", ""},
		// Partial match should NOT strip (e.g. /default should not match /de)
		{"/default", "", "/default"},
		{"/deploy", "", "/deploy"},
		{"/desktop", "", "/desktop"},
		// API paths should not be matched (they don't have lang prefixes)
		{"/api/schematics", "", "/api/schematics"},
	}
	for _, tt := range tests {
		gotLang, gotStripped := StripLangPrefix(tt.urlPath)
		if gotLang != tt.wantLang || gotStripped != tt.wantStripped {
			t.Errorf("StripLangPrefix(%q) = (%q, %q), want (%q, %q)", tt.urlPath, gotLang, gotStripped, tt.wantLang, tt.wantStripped)
		}
	}
}

func TestAllHreflangs(t *testing.T) {
	entries := AllHreflangs()
	if len(entries) != 9 {
		t.Errorf("AllHreflangs() returned %d entries, want 9", len(entries))
	}
	// Check English is first and has empty prefix
	if entries[0].HreflangCode != "en" || entries[0].Prefix != "" {
		t.Errorf("First hreflang entry: got code=%q prefix=%q, want code=en prefix=empty", entries[0].HreflangCode, entries[0].Prefix)
	}
}

func TestHreflangEntry_FullPath(t *testing.T) {
	tests := []struct {
		entry    HreflangEntry
		barePath string
		want     string
	}{
		{HreflangEntry{Lang: "en"}, "/schematics/foo", "/schematics/foo"},
		{HreflangEntry{Lang: "de", Prefix: "de"}, "/schematics/foo", "/de/schematics/foo"},
		{HreflangEntry{Lang: "zh-Hans", Prefix: "zh"}, "/", "/zh"},
	}
	for _, tt := range tests {
		got := tt.entry.FullPath(tt.barePath)
		if got != tt.want {
			t.Errorf("HreflangEntry{Lang:%q}.FullPath(%q) = %q, want %q", tt.entry.Lang, tt.barePath, got, tt.want)
		}
	}
}
