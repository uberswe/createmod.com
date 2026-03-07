package pages

import (
	"createmod/internal/i18n"
	"createmod/internal/server"
	htmltmpl "html/template"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func projectRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to determine caller file path")
	}
	// internal/pages -> project root is two levels up
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

// renderTemplate is a small helper to load and render a single template file
// with the given data using PocketBase's template registry.
func renderTemplate(t *testing.T, file string, data any) string {
	t.Helper()
	r := server.NewRegistry()
	// Register minimal func map to match production (HumanDate used in sidebar).
	r.AddFuncs(htmltmpl.FuncMap{
		"HumanDate": func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04 MST") },
		"LangFlag":  func(code string) string { return code },
		"T":         func(lang, key string) string { return i18n.T(lang, key) },
		"LangURL": func(lang string, path string) string {
			return PrefixedPath(lang, path)
		},
		"Hreflangs": func(barePath string) []HreflangEntry {
			return AllHreflangs()
		},
	})
	full := filepath.Join(projectRootFromThisFile(t), file)
	html, err := r.LoadFiles(full).Render(data)
	if err != nil {
		t.Fatalf("render failed for %s: %v", file, err)
	}
	return html
}

func Test_AuthUI_LoggedOut_HeaderAndSidebarMatch(t *testing.T) {
	d := DefaultData{IsAuthenticated: false}

	header := renderTemplate(t, "template/include/header.html", d)
	sidebar := renderTemplate(t, "template/include/sidebar.html", d)

	// Both should show a Login link and not show a Logout link
	if !(strings.Contains(header, "/login") && strings.Contains(header, "Login")) {
		t.Errorf("header (logged out) should contain Login link")
	}
	if !(strings.Contains(sidebar, "/login") && strings.Contains(sidebar, "Login")) {
		t.Errorf("sidebar (logged out) should contain Login link")
	}
	if strings.Contains(header, "/logout") {
		t.Errorf("header (logged out) should not contain /logout link")
	}
	if strings.Contains(sidebar, "/logout") {
		t.Errorf("sidebar (logged out) should not contain /logout link")
	}
}

func Test_AuthUI_LoggedIn_HeaderAndSidebarMatch(t *testing.T) {
	d := DefaultData{IsAuthenticated: true, Username: "Alice", UsernameSlug: "alice", HasAvatar: false}

	header := renderTemplate(t, "template/include/header.html", d)
	sidebar := renderTemplate(t, "template/include/sidebar.html", d)

	// Both should show Profile and Logout links when authenticated
	if !strings.Contains(header, "/profile") {
		t.Errorf("header (logged in) should contain /profile link")
	}
	if !strings.Contains(sidebar, "/profile") {
		t.Errorf("sidebar (logged in) should contain /profile link")
	}
	if !strings.Contains(header, "/logout") {
		t.Errorf("header (logged in) should contain /logout link")
	}
	if !strings.Contains(sidebar, "/logout") {
		t.Errorf("sidebar (logged in) should contain /logout link")
	}
}
