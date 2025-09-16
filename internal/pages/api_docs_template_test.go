package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_APIDocs_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "api_docs.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	must := []string{
		"API Documentation",
		"Authentication & API keys",
		"Endpoints",
		"Authorization: Bearer",
		"/api/schematics",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("api_docs.html missing %q", m)
		}
	}
}
