package pages

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/models"
)

func renderIndexWithTrending(t *testing.T, trending []models.Schematic) string {
	t.Helper()
	r := NewTestRegistry()

	files := append([]string{
		"./template/index.html",
		"./template/include/schematic_card.html",
		"./template/include/schematic_card_small.html",
	}, commonTemplates...)

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, filepath.Join(root, f))
	}

	d := IndexData{Trending: trending}
	d.DefaultData.Language = "en"
	d.Title = "Home"

	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render index.html failed: %v", err)
	}
	return out
}

func Test_Schematic_Card_Eager_Image_Loading(t *testing.T) {
	trending := make([]models.Schematic, 14)
	for i := range trending {
		trending[i] = models.Schematic{
			ID:            fmt.Sprintf("id%02d", i),
			Name:          fmt.Sprintf("name-%02d", i),
			Title:         fmt.Sprintf("Title %02d", i),
			FeaturedImage: "img.png",
			Language:      "en",
		}
		if i < 12 {
			trending[i].EagerImage = true
		}
	}

	out := renderIndexWithTrending(t, trending)

	if got := strings.Count(out, `fetchpriority="high"`); got != 12 {
		t.Fatalf("expected 12 fetchpriority=high thumbnails, got %d", got)
	}
	// Each eager card has two eager imgs (placeholder + thumbnail).
	if got := strings.Count(out, `loading="eager"`); got != 24 {
		t.Fatalf("expected 24 eager-loading imgs, got %d", got)
	}
	// The remaining 2 cards stay lazy (2 imgs each).
	if got := strings.Count(out, `loading="lazy"`); got != 4 {
		t.Fatalf("expected 4 lazy-loading imgs, got %d", got)
	}
}

func Test_Schematic_Card_Thumb_URLs_Versioned(t *testing.T) {
	trending := []models.Schematic{{
		ID:            "id01",
		Name:          "name-01",
		Title:         "Title 01",
		FeaturedImage: "img.png",
		Language:      "en",
	}}

	out := renderIndexWithTrending(t, trending)

	if !strings.Contains(out, "?thumb=320x180&v=2") {
		t.Fatalf("expected thumbnail URLs to carry the v=2 cache-bust parameter")
	}
	if strings.Contains(out, `?thumb=320x180"`) {
		t.Fatalf("found unversioned thumbnail URL; all thumb URLs must include v=2")
	}
}
