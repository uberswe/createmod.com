package pages

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	pbtempl "github.com/pocketbase/pocketbase/tools/template"
)

// Reuse the helper to find project root (duplicated here to keep test self-contained).
func projectRootFromThisFile_search(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to determine caller file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func renderTemplate_search(t *testing.T, file string, data any) string {
	t.Helper()
	r := pbtempl.NewRegistry()
	full := filepath.Join(projectRootFromThisFile_search(t), file)
	html, err := r.LoadFiles(full).Render(data)
	if err != nil {
		t.Fatalf("render failed for %s: %v", file, err)
	}
	return html
}

// Test that the header search form has the expected HTMX attributes
func Test_Search_HTMX_HeaderFormAttributes(t *testing.T) {
	d := DefaultData{IsAuthenticated: false}
	header := renderTemplate_search(t, "template/include/header.html", d)

	// The form should have hx-post="/search"
	if !strings.Contains(header, "hx-post=\"/search\"") {
		t.Errorf("header search form should include hx-post=\"/search\"")
	}
	// It should target body outerHTML and push URL
	if !strings.Contains(header, "hx-target=\"body\"") {
		t.Errorf("header search form should include hx-target=\"body\"")
	}
	if !strings.Contains(header, "hx-swap=\"outerHTML\"") {
		t.Errorf("header search form should include hx-swap=\"outerHTML\"")
	}
	if !strings.Contains(header, "hx-push-url=\"true\"") {
		t.Errorf("header search form should include hx-push-url=\"true\"")
	}
}
