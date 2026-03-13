package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Download_Interstitial_Template_Has_Countdown_And_Links(t *testing.T) {
	path := filepath.Join("..", "..", "template", "download_interstitial.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	// Check free download elements
	must := []string{
		"Preparing your download",
		"id=\"countdown\"",
		"id=\"token-id\"",
		"/api/download-url/",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("download_interstitial.html missing %q", m)
		}
	}
	// Check paid external elements
	must2 := []string{
		"Preparing external link",
		"id=\"countdown-ext\"",
		"SignedOutURL .ExternalURL",
	}
	for _, m := range must2 {
		if !strings.Contains(s, m) {
			t.Fatalf("download_interstitial.html missing %q (paid)", m)
		}
	}
}
