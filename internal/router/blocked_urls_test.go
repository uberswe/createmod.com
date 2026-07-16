package router

import (
	"net/http/httptest"
	"testing"
)

// LangPrefixHandler strips the language prefix before this middleware runs,
// so effectiveRequestURL must restore it for blocked entries stored with a
// prefix (e.g. /es/search?q=foo) to ever match.
func Test_effectiveRequestURL(t *testing.T) {
	tests := []struct {
		name string
		path string
		lang string
		want string
	}{
		{"no lang header", "/search?q=foo", "", "/search?q=foo"},
		{"es prefix restored", "/search?p=17&q=canva", "es", "/es/search?p=17&q=canva"},
		{"unknown lang left alone", "/search?q=foo", "xx", "/search?q=foo"},
		{"pt-BR prefix restored", "/schematics/some-build", "pt-BR", "/pt-br/schematics/some-build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.lang != "" {
				req.Header.Set("X-Createmod-Lang", tt.lang)
			}
			got := effectiveRequestURL(req)
			if got.RequestURI() != tt.want {
				t.Fatalf("effectiveRequestURL(%q, lang=%q) = %q, want %q", tt.path, tt.lang, got.RequestURI(), tt.want)
			}
		})
	}
}
