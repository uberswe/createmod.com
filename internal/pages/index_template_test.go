package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Index_Template_Shows_Sections(t *testing.T) {
	path := filepath.Join("..", "..", "template", "index.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if !strings.Contains(s, "Trending") {
		t.Fatalf("index.html should contain 'Trending' heading")
	}
	if !strings.Contains(s, "Latest") {
		t.Fatalf("index.html should contain 'Latest' heading")
	}
	if !strings.Contains(s, "Highest Rated") {
		t.Fatalf("index.html should contain 'Highest Rated' heading")
	}
	if !strings.Contains(s, "Categories") {
		t.Fatalf("index.html should contain 'Categories' heading")
	}
	// Ensure sections use the small card template
	if !strings.Contains(s, `template "schematic_card_small.html"`) {
		t.Fatalf("index.html should use schematic_card_small.html template")
	}
}

func Test_Index_Template_No_Featured(t *testing.T) {
	path := filepath.Join("..", "..", "template", "index.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)

	if strings.Contains(s, "Featured Builds") {
		t.Fatalf("index.html should not contain 'Featured Builds' section")
	}
	if strings.Contains(s, `template "schematic_card_featured.html"`) {
		t.Fatalf("index.html should not use schematic_card_featured.html template")
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
