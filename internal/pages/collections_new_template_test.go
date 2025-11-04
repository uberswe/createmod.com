package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Collections_New_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "collections_new.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Create collection",
		"action=\"/collections\"",
		"method=\"post\"",
		"enctype=\"multipart/form-data\"",
		"name=\"title\"",
		"name=\"description\"",
		"name=\"banner_url\"",
		"name=\"banner\"",
		"accept=\"image/png,image/jpeg,image/webp\"",
		"Recommended 1600x400 (4:1), max 2MB",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("collections_new.html missing: %s", m)
		}
	}
}
