package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Index_Template_Shows_Trending_First(t *testing.T) {
	path := filepath.Join("..", "..", "template", "index.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "Trending") {
		t.Fatalf("index.html should contain 'Trending' tab")
	}
	if !strings.Contains(s, "Highest Rated") {
		t.Fatalf("index.html should contain 'Highest Rated' tab")
	}
	if !strings.Contains(s, "Featured Builds") {
		t.Fatalf("index.html should contain 'Featured Builds' section")
	}
	// Ensure tabbed sections use the small card template
	if !strings.Contains(s, `template "schematic_card_small.html"`) {
		t.Fatalf("index.html should use schematic_card_small.html template")
	}
	// Ensure featured section uses the featured card template
	if !strings.Contains(s, `template "schematic_card_featured.html"`) {
		t.Fatalf("index.html should use schematic_card_featured.html template")
	}
}

func Test_Index_Template_Removed_Tags_Section(t *testing.T) {
	path := filepath.Join("..", "..", "template", "index.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if strings.Contains(s, "Popular Schematic Tags") {
		t.Fatalf("index.html should not contain the 'Popular Schematic Tags' section anymore")
	}
	if strings.Contains(s, `{{range .Tags }}`) {
		t.Fatalf("index.html should not iterate over .Tags on the main page anymore")
	}
}
