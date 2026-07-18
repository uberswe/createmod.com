package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_AdminBlockedURLs_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "admin_blocked_urls.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Admin: Blocked URLs",
		"Block a URL",
		`name="url"`,
		`name="note"`,
		`action="/admin/blocked-urls"`,
		"/admin/blocked-urls/{{ .ID }}/delete",
		"Unblock",
		"No blocked URLs.",
		"admin_nav.html",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("admin_blocked_urls.html missing: %s", m)
		}
	}
}

func Test_AdminNav_Has_BlockedURLs_Link(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "admin_nav.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(b), `href="/admin/blocked-urls"`) {
		t.Fatal("admin_nav.html missing blocked URLs link")
	}
}
