package pages

import (
	"createmod/internal/models"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	r := NewTestRegistry()
	full := filepath.Join(projectRootFromThisFile_search(t), file)
	html, err := r.LoadFiles(full).Render(data)
	if err != nil {
		t.Fatalf("render failed for %s: %v", file, err)
	}
	return html
}

func renderSearchPage(t *testing.T, data SearchData) string {
	t.Helper()
	root := projectRootFromThisFile_search(t)
	r := NewTestRegistry()
	files := make([]string, len(searchTemplates))
	for i, f := range searchTemplates {
		files[i] = filepath.Join(root, f)
	}
	html, err := r.LoadFiles(files...).Render(data)
	if err != nil {
		t.Fatalf("render search page failed: %v", err)
	}
	return html
}

// Test that the header search form has the expected HTMX attributes
func Test_Search_HTMX_HeaderFormAttributes(t *testing.T) {
	d := DefaultData{IsAuthenticated: false}
	header := renderTemplate_search(t, "template/include/header.html", d)

	if !strings.Contains(header, "hx-post=\"/search\"") {
		t.Errorf("header search form should include hx-post=\"/search\"")
	}
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

func Test_Search_Template_Has_Filter_Sidebar(t *testing.T) {
	d := SearchData{
		Sort:     6,
		Rating:   -1,
		Category: "all",
		Tag:      "all",
		ViewMode: "grid",
	}
	html := renderSearchPage(t, d)
	if !strings.Contains(html, "search-filters") {
		t.Error("search page should contain search-filters id")
	}
	// The old collapseAdvanced should not be present
	if strings.Contains(html, "collapseAdvanced") {
		t.Error("search page should not contain collapseAdvanced (filters are always visible)")
	}
}

func Test_Search_Template_Has_Trending_Sort(t *testing.T) {
	d := SearchData{
		Sort:     8,
		Rating:   -1,
		Category: "all",
		Tag:      "all",
		ViewMode: "grid",
	}
	html := renderSearchPage(t, d)
	if !strings.Contains(html, `value="8"`) {
		t.Error("search page should have trending sort option with value=8")
	}
	if !strings.Contains(html, "Trending") {
		t.Error("search page should have 'Trending' label")
	}
}

func Test_Search_Template_Has_Hero_Input(t *testing.T) {
	d := SearchData{
		Sort:     6,
		Rating:   -1,
		Category: "all",
		Tag:      "all",
		ViewMode: "grid",
	}
	html := renderSearchPage(t, d)
	if !strings.Contains(html, "search-hero-input") {
		t.Error("search page should contain search-hero-input")
	}
	if !strings.Contains(html, `hx-get="/search"`) {
		t.Error("search hero input should have hx-get for live search")
	}
}

func Test_Search_Card_Shows_Author(t *testing.T) {
	root := projectRootFromThisFile_search(t)
	r := NewTestRegistry()
	cardFile := filepath.Join(root, "template/include/schematic_card.html")
	data := models.Schematic{
		ID:            "test-id",
		Title:         "Test Schematic",
		Name:          "test-schematic",
		FeaturedImage: "test.jpg",
		Author:        &models.User{Username: "TestUser"},
		Views:         42,
	}
	html, err := r.LoadFiles(cardFile).Render(data)
	if err != nil {
		t.Fatalf("render card failed: %v", err)
	}
	if !strings.Contains(html, "TestUser") {
		t.Error("schematic card should show author username")
	}
}

func Test_Search_Card_Shows_Tags(t *testing.T) {
	root := projectRootFromThisFile_search(t)
	r := NewTestRegistry()
	cardFile := filepath.Join(root, "template/include/schematic_card.html")
	data := models.Schematic{
		ID:            "test-id",
		Title:         "Test Schematic",
		Name:          "test-schematic",
		FeaturedImage: "test.jpg",
		Author:        &models.User{Username: "TestUser"},
		HasTags:       true,
		Tags: []models.SchematicTag{
			{Name: "Redstone"},
			{Name: "Farm"},
			{Name: "Compact"},
			{Name: "FourthTag"},
		},
		Views: 42,
	}
	html, err := r.LoadFiles(cardFile).Render(data)
	if err != nil {
		t.Fatalf("render card failed: %v", err)
	}
	if !strings.Contains(html, "Redstone") {
		t.Error("schematic card should show first tag")
	}
	if !strings.Contains(html, "Farm") {
		t.Error("schematic card should show second tag")
	}
	if !strings.Contains(html, "Compact") {
		t.Error("schematic card should show third tag")
	}
	// Fourth tag should NOT appear (limit to 3)
	if strings.Contains(html, "FourthTag") {
		t.Error("schematic card should not show more than 3 tags")
	}
}
