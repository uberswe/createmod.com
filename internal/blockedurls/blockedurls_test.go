package blockedurls

import (
	"net/url"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"full url with query", "https://createmod.com/es/search?p=17&q=canva-premium-mod-for-pc", "/es/search?p=17&q=canva-premium-mod-for-pc", false},
		{"absolute path with query", "/es/search?q=foo", "/es/search?q=foo", false},
		{"path only", "/schematics/some-slug", "/schematics/some-slug", false},
		{"surrounding whitespace", "  https://createmod.com/foo?a=1  ", "/foo?a=1", false},
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"relative path", "search?q=foo", "", true},
		{"bare domain no path", "https://createmod.com", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	entries := []string{
		"/es/search?p=17&q=canva-premium-mod-for-pc",
		"/schematics/blocked-build",
	}
	tests := []struct {
		name string
		req  string
		want bool
	}{
		{"exact match", "/es/search?p=17&q=canva-premium-mod-for-pc", true},
		{"query order swapped", "/es/search?q=canva-premium-mod-for-pc&p=17", true},
		{"different page param", "/es/search?p=18&q=canva-premium-mod-for-pc", false},
		{"missing param", "/es/search?q=canva-premium-mod-for-pc", false},
		{"extra param", "/es/search?p=17&q=canva-premium-mod-for-pc&x=1", false},
		{"different path", "/search?p=17&q=canva-premium-mod-for-pc", false},
		{"path-only entry match", "/schematics/blocked-build", true},
		{"path-only entry with query", "/schematics/blocked-build?utm=1", false},
		{"unrelated path", "/schematics/other-build", false},
		{"search page without query", "/es/search", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			if got := Matches(entries, u); got != tt.want {
				t.Fatalf("Matches(%q) = %v, want %v", tt.req, got, tt.want)
			}
		})
	}
}

func TestMatchesEntryWithHost(t *testing.T) {
	// Entries stored as full URLs (pre-normalization data) still match on
	// path + query.
	entries := []string{"https://createmod.com/es/search?q=foo"}
	u, _ := url.Parse("/es/search?q=foo")
	if !Matches(entries, u) {
		t.Fatal("expected full-URL entry to match on path + query")
	}
}
