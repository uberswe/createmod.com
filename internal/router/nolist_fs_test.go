package router

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// TestNoListFS verifies directory listings are disabled: a directory with no
// index.html is not openable (404 via http.FileServer), while files and
// directories that do have an index.html still work.
func TestNoListFS(t *testing.T) {
	root := t.TempDir()
	// root/ads/banner.webp  (a directory with no index.html)
	if err := os.MkdirAll(filepath.Join(root, "ads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ads", "banner.webp"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	// root/withindex/index.html
	if err := os.MkdirAll(filepath.Join(root, "withindex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "withindex", "index.html"), []byte("<h1>ok</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := noListFS{http.Dir(root)}

	// A directory without index.html must be denied (-> 404).
	if _, err := fs.Open("/ads"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open(/ads) err = %v, want os.ErrNotExist (listing not disabled)", err)
	}
	if _, err := fs.Open("/ads/"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open(/ads/) err = %v, want os.ErrNotExist", err)
	}

	// A real file must still serve.
	f, err := fs.Open("/ads/banner.webp")
	if err != nil {
		t.Errorf("Open(/ads/banner.webp) err = %v, want nil", err)
	} else {
		f.Close()
	}

	// A directory that has an index.html is allowed (FileServer serves it).
	d, err := fs.Open("/withindex")
	if err != nil {
		t.Errorf("Open(/withindex) err = %v, want nil (has index.html)", err)
	} else {
		d.Close()
	}
}
