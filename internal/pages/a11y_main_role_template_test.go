package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Guides_Template_Has_Main_Role(t *testing.T) {
	path := filepath.Join("..", "..", "template", "guides.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.Contains(s, "<main") {
		t.Fatalf("guides.html should use a <main> element for the main content wrapper")
	}
}

func Test_Profile_Template_Has_Main_Role(t *testing.T) {
	path := filepath.Join("..", "..", "template", "profile.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.Contains(s, "<main") {
		t.Fatalf("profile.html should use a <main> element for the main content wrapper")
	}
}
