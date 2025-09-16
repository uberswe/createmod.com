package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Collections_Show_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "collections_show.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"page-title",
		"Back to collections",
		"Collection details",
		"Featured",
		"Download all",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("collections_show.html missing: %s", m)
		}
	}
}
