package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func Test_Header_Search_HTMX_Attributes(t *testing.T) {
	// Ensure header search form retains our HTMX navigation contract
	path := filepath.Join("..", "..", "template", "include", "header.html")
	s := mustRead(t, path)

	attrs := []string{
		`hx-post="/search"`,
		`hx-target="body"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
	}
	for _, a := range attrs {
		if !strings.Contains(s, a) {
			t.Fatalf("header.html missing attribute: %s", a)
		}
	}
}

func Test_Header_Has_Logout_Link(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "header.html")
	s := mustRead(t, path)
	if !strings.Contains(s, `/logout`) {
		t.Fatalf("header.html expected to contain /logout reference")
	}
}

func Test_Sidebar_Has_Logout_And_Profile_Links(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "sidebar.html")
	s := mustRead(t, path)
	if !strings.Contains(s, `/logout`) {
		t.Fatalf("sidebar.html expected to contain /logout reference")
	}
	if !strings.Contains(s, `/profile`) {
		t.Fatalf("sidebar.html expected to contain /profile reference")
	}
}
