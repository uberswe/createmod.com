package pages

import (
	"path/filepath"
	"strings"
	"testing"
)

func Test_Collections_Show_Template_Placeholder_When_No_Banner(t *testing.T) {
	r := NewTestRegistry()

	files := append([]string{
		"./template/collections_show.html",
	}, commonTemplates...)

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, filepath.Join(root, f))
	}

	d := CollectionsShowData{}
	d.DefaultData.Language = "en"
	d.Title = "Collection"
	d.Description = ""
	d.BannerURL = ""

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render collections_show.html failed: %v", err)
	}
	// contains placeholder message
	if !containsAll(out, []string{"No banner image set", "Recommended size 1600x400"}) {
		t.Fatalf("expected placeholder banner hint when no banner is set")
	}
}

// containsAll is a small helper used only in this test file.
func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
