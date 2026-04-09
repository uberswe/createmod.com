package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Videos_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "videos.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Videos",
		"SignedOutURL",
		"View on YouTube",
		"View schematic",
		"videos-sticky-adrail",
		"name=\"q\"",
		"hx-target=\"#videos-results\"",
		"hx-select=\"#videos-results\"",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("videos.html missing: %s", m)
		}
	}
}
