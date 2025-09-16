package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Profile_Template_Contributor_Badge(t *testing.T) {
	path := filepath.Join("..", "..", "template", "profile.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	// Ensure we render a Contributor badge conditionally when user has schematics
	if !strings.Contains(s, "Contributor") {
		t.Fatalf("profile.html should contain a 'Contributor' badge marker")
	}
}
