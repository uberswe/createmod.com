package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Header_Template_Contains_Breadcrumb_Markup(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "header.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	must := []string{
		`aria-label="breadcrumb"`,
		"breadcrumb-item",
		".Breadcrumbs",
		".BreadcrumbJSONLD",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("header.html missing %q", m)
		}
	}
}

func Test_Header_Template_Breadcrumbs_Conditional(t *testing.T) {
	path := filepath.Join("..", "..", "template", "include", "header.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	// Breadcrumbs should be wrapped in {{ if .Breadcrumbs }}
	if !strings.Contains(s, "{{ if .Breadcrumbs }}") {
		t.Fatal("breadcrumb block should be conditional on .Breadcrumbs")
	}
}
