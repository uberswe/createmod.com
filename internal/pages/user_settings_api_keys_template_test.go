package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_User_Settings_Template_Has_API_Keys_Section(t *testing.T) {
	path := filepath.Join("..", "..", "template", "user-api-keys.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"API Keys",
		"/settings/api-keys/new",
		"Generate new key",
		"/settings/api-keys/{{ .ID }}/revoke",
		"Revoke",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("user-settings.html missing: %q", m)
		}
	}
}
