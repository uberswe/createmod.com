package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_External_Interstitial_Template_Has_Warning_Countdown_And_Link(t *testing.T) {
	path := filepath.Join("..", "..", "template", "external_interstitial.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"You are leaving createmod.com",
		"id=\"ext-countdown\"",
		"id=\"ext-continue\"",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("external_interstitial.html missing %q", m)
		}
	}
}
