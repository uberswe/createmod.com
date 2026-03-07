package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Profile_Template_Usage_Section(t *testing.T) {
	path := filepath.Join("..", "..", "template", "profile.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Usage",
		"Total schematics",
		"Total views",
		"Total downloads",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("profile.html missing %q", m)
		}
	}
}
