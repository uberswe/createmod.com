package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_UploadPreview_Template_Has_Scheduling_Input(t *testing.T) {
	path := filepath.Join("..", "..", "template", "upload_preview.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"name=\"scheduled_at\"",
		"type=\"datetime-local\"",
		"Schedule publish time (optional)",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("upload_preview.html missing: %s", m)
		}
	}
}
