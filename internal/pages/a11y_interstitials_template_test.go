package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Download_Interstitial_Template_A11y(t *testing.T) {
	path := filepath.Join("..", "..", "template", "download_interstitial.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"role=\"main\"",
		"id=\"countdown\" aria-live=\"polite\"",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("download_interstitial.html missing a11y marker: %q", m)
		}
	}
}

func Test_External_Interstitial_Template_A11y(t *testing.T) {
	path := filepath.Join("..", "..", "template", "external_interstitial.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"role=\"main\"",
		"id=\"ext-countdown\" aria-live=\"polite\"",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("external_interstitial.html missing a11y marker: %q", m)
		}
	}
}
