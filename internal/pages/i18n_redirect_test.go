package pages

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"createmod/internal/server"
)

// TestRedirectToPreferredLang verifies that bare-path requests carrying a
// non-English cm_lang cookie are redirected to the language-prefixed URL with
// uncacheable headers, while English / already-prefixed / non-GET requests are
// left to render normally. This is the core of the CDN cross-language fix:
// every cacheable URL must be bound to a single language.
func TestRedirectToPreferredLang(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		rawQuery     string
		cookie       string // cm_lang cookie value, "" for none
		xLangHeader  string // X-Createmod-Lang, set by LangPrefixHandler for prefixed paths
		wantRedirect bool
		wantLocation string
	}{
		{
			name:         "ru cookie on bare root redirects to /ru/",
			method:       http.MethodGet,
			path:         "/",
			cookie:       "ru",
			wantRedirect: true,
			wantLocation: "/ru",
		},
		{
			name:         "ru cookie on bare schematic detail redirects preserving path",
			method:       http.MethodGet,
			path:         "/schematics/cool-build",
			cookie:       "ru",
			wantRedirect: true,
			wantLocation: "/ru/schematics/cool-build",
		},
		{
			name:         "query string is preserved",
			method:       http.MethodGet,
			path:         "/schematics",
			rawQuery:     "p=2",
			cookie:       "zh-Hans",
			wantRedirect: true,
			wantLocation: "/zh/schematics?p=2",
		},
		{
			name:         "english cookie does not redirect",
			method:       http.MethodGet,
			path:         "/",
			cookie:       "en",
			wantRedirect: false,
		},
		{
			name:         "no cookie does not redirect (canonical english)",
			method:       http.MethodGet,
			path:         "/",
			wantRedirect: false,
		},
		{
			name:         "unsupported cookie does not redirect",
			method:       http.MethodGet,
			path:         "/",
			cookie:       "de",
			wantRedirect: false,
		},
		{
			name:         "already prefixed path does not redirect",
			method:       http.MethodGet,
			path:         "/schematics",
			cookie:       "ru",
			xLangHeader:  "ru",
			wantRedirect: false,
		},
		{
			name:         "POST is not redirected",
			method:       http.MethodPost,
			path:         "/",
			cookie:       "ru",
			wantRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.path
			if tt.rawQuery != "" {
				target += "?" + tt.rawQuery
			}
			r := httptest.NewRequest(tt.method, target, nil)
			if tt.cookie != "" {
				r.AddCookie(&http.Cookie{Name: "cm_lang", Value: tt.cookie})
			}
			if tt.xLangHeader != "" {
				r.Header.Set("X-Createmod-Lang", tt.xLangHeader)
			}
			w := httptest.NewRecorder()
			e := server.NewRequestEvent(w, r)

			redirected, err := RedirectToPreferredLang(e)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if redirected != tt.wantRedirect {
				t.Fatalf("redirected = %v, want %v", redirected, tt.wantRedirect)
			}
			if !tt.wantRedirect {
				return
			}
			if got := w.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
			if got := w.Code; got != http.StatusFound {
				t.Errorf("status = %d, want %d", got, http.StatusFound)
			}
			// The redirect is cookie-dependent and must never be shared-cached.
			if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
				t.Errorf("Cache-Control = %q, want %q", got, "private, no-store")
			}
		})
	}
}
