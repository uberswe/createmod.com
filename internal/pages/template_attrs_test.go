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

func Test_Header_Has_Search_Button(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "header.html")
	s := mustRead(t, path)

	if !strings.Contains(s, `btn-search`) {
		t.Fatalf("header.html missing search button with btn-search class")
	}
	if !strings.Contains(s, `/search`) {
		t.Fatalf("header.html missing link to /search")
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
