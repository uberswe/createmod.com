package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Guides_New_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "guides_new.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"New Guide",
		"action=\"/guides\"",
		"name=\"title\"",
		"name=\"content\"",
		"name=\"video_url\"",
		"name=\"external_url\"",
		"guide-editor",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("guides_new.html missing: %q", m)
		}
	}
}
