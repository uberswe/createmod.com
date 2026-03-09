package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Guides_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "guides.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Guides",
		"name=\"q\"",
		"hx-target=\"#guides-results\"",
		"hx-select=\"#guides-results\"",
		"Read guide",
		"card-link",
		"guides-adrail",
		"Views",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("guides.html missing: %s", m)
		}
	}
}
