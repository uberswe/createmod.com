package pages

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_IsSupportedLanguage(t *testing.T) {
	cases := map[string]bool{
		"en":      true,
		"pt-BR":   true,
		"pt-PT":   true,
		"es":      true,
		"de":      true,
		"pl":      true,
		"ru":      true,
		"zh-Hans": true,
		"xx":      false,
		"":        false,
	}
	for code, want := range cases {
		if got := isSupportedLanguage(code); got != want {
			t.Fatalf("isSupportedLanguage(%q) = %v; want %v", code, got, want)
		}
	}
}

func Test_NormalizeFromAcceptLanguage(t *testing.T) {
	tests := []struct{ in, want string }{
		{"pt-BR", "pt-BR"},
		{"pt-PT", "pt-PT"},
		{"pt", "pt-PT"},
		{"pt-xx", "pt-PT"},
		{"es-ES,es;q=0.9", "es"},
		{"DE-de", "de"},
		{"pl-PL", "pl"},
		{"ru-RU", "ru"},
		{"zh-CN", "zh-Hans"},
		{"fr-FR", "en"},
		{"", "en"},
	}
	for _, tc := range tests {
		if got := normalizeFromAcceptLanguage(tc.in); got != tc.want {
			t.Fatalf("normalizeFromAcceptLanguage(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func Test_PreferredLanguageFromRequest(t *testing.T) {
	// Cookie takes precedence when supported
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "cm_lang", Value: "de"})
	if got := preferredLanguageFromRequest(req); got != "de" {
		t.Fatalf("preferredLanguageFromRequest with de cookie = %q; want de", got)
	}

	// Unsupported cookie falls back to Accept-Language
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(&http.Cookie{Name: "cm_lang", Value: "xx"})
	req2.Header.Set("Accept-Language", "es-ES")
	if got := preferredLanguageFromRequest(req2); got != "es" {
		t.Fatalf("preferredLanguageFromRequest with invalid cookie + es header = %q; want es", got)
	}

	// No cookie, zh header
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set("Accept-Language", "zh-HK")
	if got := preferredLanguageFromRequest(req3); got != "zh-Hans" {
		t.Fatalf("preferredLanguageFromRequest zh-HK = %q; want zh-Hans", got)
	}

	// No cookie, no header
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := preferredLanguageFromRequest(req4); got != "en" {
		t.Fatalf("preferredLanguageFromRequest default = %q; want en", got)
	}
}
