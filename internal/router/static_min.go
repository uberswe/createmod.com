package router

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

// Static CSS/JS under /assets/x/ is minified on first request and cached in
// memory (keyed by file mod time), so source files stay readable in the repo
// without a frontend build step.

// minifySkip lists assets served verbatim: editor.js is already minified by
// esbuild, and re-minifying a 1.4 MB bundle adds startup cost for no gain.
var minifySkip = map[string]bool{
	"editor.js": true,
}

type minifiedAsset struct {
	data    []byte
	modTime time.Time
}

var (
	assetMinifier  *minify.M
	minifiedAssets sync.Map // clean path -> *minifiedAsset
)

func init() {
	assetMinifier = minify.New()
	assetMinifier.AddFunc("text/css", css.Minify)
	assetMinifier.AddFunc("application/javascript", js.Minify)
}

// serveMinifiedAsset serves .css/.js files from root minified. Returns false
// when the request should fall through to the plain file server (non-CSS/JS,
// skiplisted, or unreadable). Minification failures fail open to the raw
// bytes so a syntax quirk can never take an asset down.
func serveMinifiedAsset(w http.ResponseWriter, req *http.Request, root string) bool {
	name := strings.TrimPrefix(req.URL.Path, "/assets/x/")
	var mediatype string
	switch path.Ext(name) {
	case ".css":
		mediatype = "text/css"
	case ".js":
		mediatype = "application/javascript"
	default:
		return false
	}
	if minifySkip[path.Base(name)] {
		return false
	}
	clean := path.Clean("/" + name) // forbids .. traversal
	fsPath := root + clean
	fi, err := os.Stat(fsPath)
	if err != nil || fi.IsDir() {
		return false
	}
	if cached, ok := minifiedAssets.Load(clean); ok {
		ca := cached.(*minifiedAsset)
		if ca.modTime.Equal(fi.ModTime()) {
			w.Header().Set("Content-Type", mediatype+"; charset=utf-8")
			http.ServeContent(w, req, name, ca.modTime, bytes.NewReader(ca.data))
			return true
		}
	}
	raw, err := os.ReadFile(fsPath)
	if err != nil {
		return false
	}
	out, err := assetMinifier.Bytes(mediatype, raw)
	if err != nil {
		out = raw
	}
	ca := &minifiedAsset{data: out, modTime: fi.ModTime()}
	minifiedAssets.Store(clean, ca)
	w.Header().Set("Content-Type", mediatype+"; charset=utf-8")
	http.ServeContent(w, req, name, ca.modTime, bytes.NewReader(ca.data))
	return true
}
