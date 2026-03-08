package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Collections_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "collections.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Collections",
		"name=\"q\"",
		"hx-target=\"#collections-results\"",
		"hx-select=\"#collections-results\"",
		"View collection",
		"Featured",
		"nav-tabs",
		"Public",
		"My Collections",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("collections.html missing: %s", m)
		}
	}
}
