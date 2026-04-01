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

func Test_Header_Search_Uses_Boost(t *testing.T) {
	// The header search forms must NOT have explicit hx-post/hx-target/hx-swap
	// attributes.  They rely on the global hx-boost="true" on <body> to handle
	// POST → redirect → full-page swap correctly.  Explicit attributes cause a
	// broken body-innerHTML swap that loses <head> scripts and body attributes.
	path := filepath.Join("..", "..", "template", "include", "header.html")
	s := mustRead(t, path)

	forbidden := []string{
		`hx-post="/search"`,
		`hx-target="body"`,
		`hx-swap="innerHTML"`,
		`hx-select="body"`,
	}
	for _, a := range forbidden {
		if strings.Contains(s, a) {
			t.Fatalf("header.html must not contain %s — rely on hx-boost instead", a)
		}
	}

	// The forms must still have action="/search" method="post" for boosting.
	if !strings.Contains(s, `action="/search"`) {
		t.Fatalf("header.html missing action=\"/search\"")
	}
	if !strings.Contains(s, `method="post"`) {
		t.Fatalf("header.html missing method=\"post\"")
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
