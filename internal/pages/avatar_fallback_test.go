package pages

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Reuse the helper from auth_ui_test.go if present; otherwise define a minimal file loader.
func projectRootFromThisFile_avatar(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to determine caller file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func Test_Header_Avatar_Fallback_Shows_Default(t *testing.T) {
	d := DefaultData{IsAuthenticated: true, Username: "Alice", HasAvatar: false}
	html := renderTemplate(t, "template/include/header.html", d)
	if !strings.Contains(html, "https://mc-heads.net/avatar/Steve") {
		t.Fatalf("header should contain default mc-heads avatar when HasAvatar=false")
	}
}

func Test_Sidebar_Avatar_Fallback_Shows_Default(t *testing.T) {
	d := DefaultData{IsAuthenticated: true, Username: "Alice", HasAvatar: false}
	html := renderTemplate(t, "template/include/sidebar.html", d)
	if !strings.Contains(html, "https://mc-heads.net/avatar/Steve") {
		t.Fatalf("sidebar should contain default mc-heads avatar when HasAvatar=false")
	}
}

func Test_Comments_Template_Includes_Default_Avatar_URL(t *testing.T) {
	// Basic string presence check (template-only) similar to other template tests
	path := filepath.Join(projectRootFromThisFile_avatar(t), "template", "include", "comments.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.Contains(s, "https://mc-heads.net/avatar/Steve") {
		t.Fatalf("comments include should include default mc-heads avatar url")
	}
}
