package pages

import (
	"path/filepath"
	"strings"
	"testing"
)

func Test_Collections_Show_Template_Hides_Banner_When_Empty(t *testing.T) {
	r := NewTestRegistry()

	files := append([]string{
		"./template/collections_show.html",
		"./template/include/schematic_card.html",
		"./template/include/schematic_card_small.html",
	}, commonTemplates...)

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, filepath.Join(root, f))
	}

	d := CollectionsShowData{}
	d.DefaultData.Language = "en"
	d.TitleText = "My Collection"
	d.Title = "My Collection"
	d.DescriptionText = ""
	d.BannerURL = ""

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render collections_show.html failed: %v", err)
	}
	// Banner and description sections should be hidden when empty
	if strings.Contains(out, "Collection banner") {
		t.Fatalf("expected banner image to be hidden when BannerURL is empty")
	}
	if strings.Contains(out, "Description</h3>") {
		t.Fatalf("expected description card to be hidden when DescriptionText is empty")
	}
	// Title should still render
	if !strings.Contains(out, "My Collection") {
		t.Fatalf("expected collection title to be shown")
	}
}

func Test_Collections_Show_Template_Shows_Banner_And_Description(t *testing.T) {
	r := NewTestRegistry()

	files := append([]string{
		"./template/collections_show.html",
		"./template/include/schematic_card.html",
		"./template/include/schematic_card_small.html",
	}, commonTemplates...)

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, filepath.Join(root, f))
	}

	d := CollectionsShowData{}
	d.DefaultData.Language = "en"
	d.TitleText = "Test Collection"
	d.Title = "Test Collection"
	d.DescriptionText = "A test description"
	d.DescriptionHTML = "A test description"
	d.BannerURL = "https://example.com/banner.webp"

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render collections_show.html failed: %v", err)
	}
	if !strings.Contains(out, "Collection banner") {
		t.Fatalf("expected banner image when BannerURL is set")
	}
	if !strings.Contains(out, "A test description") {
		t.Fatalf("expected description text when DescriptionText is set")
	}
}

func Test_Collections_Show_Template_Edit_Button_For_Owner(t *testing.T) {
	r := NewTestRegistry()

	files := append([]string{
		"./template/collections_show.html",
		"./template/include/schematic_card.html",
		"./template/include/schematic_card_small.html",
	}, commonTemplates...)

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, filepath.Join(root, f))
	}

	d := CollectionsShowData{}
	d.DefaultData.Language = "en"
	d.TitleText = "My Collection"
	d.Title = "My Collection"
	d.Slug = "/collections/test-slug"
	d.IsOwner = true

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render collections_show.html failed: %v", err)
	}
	if !strings.Contains(out, "/collections/test-slug/edit") {
		t.Fatalf("expected edit link for owner")
	}

	// Non-owner should not see edit button
	d.IsOwner = false
	out, err = r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render collections_show.html failed: %v", err)
	}
	if strings.Contains(out, "/edit") {
		t.Fatalf("non-owner should not see edit link")
	}
}
