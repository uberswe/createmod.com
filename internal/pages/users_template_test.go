package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Users_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "users.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Users",
		"id=\"users-results\"",
		"hx-target=\"#users-results\"",
		"hx-select=\"#users-results\"",
		"Page ",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("users.html missing: %s", m)
		}
	}
}
