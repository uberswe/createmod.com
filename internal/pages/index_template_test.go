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

	if !strings.Contains(s, "Trending Schematics") {
		t.Fatalf("index.html should contain 'Trending Schematics' section")
	}
	if !strings.Contains(s, "Highest Rated Schematics") {
		t.Fatalf("index.html should contain 'Highest Rated Schematics' section")
	}
	// Ensure the trending section (left column) appears before the main recent list that uses medium cards
	idxTrending := strings.Index(s, "Trending Schematics")
	idxRecentCards := strings.Index(s, `template "schematic_card_medium.html"`)
	if idxTrending < 0 || idxRecentCards < 0 {
		t.Fatalf("failed to locate expected markers in index.html")
	}
	if idxTrending > idxRecentCards {
		t.Fatalf("expected 'Trending Schematics' to appear before medium card list on the page")
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
