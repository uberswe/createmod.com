package testutil

import (
	"createmod/internal/auth"
	"createmod/internal/nbtparser"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Minimal view model for templates in tests
// Mirrors only the fields needed by header.html and sidebar.html
// to avoid importing the pages package in tests.
type viewData struct {
	IsAuthenticated bool
	Username        string
	UsernameSlug    string
	HasAvatar       bool
	Avatar          string
	SubCategory     string
	Title           string
	Categories      any
}

type tempUpload struct {
	Filename      string
	Size          int64
	Checksum      string
	UploadedAt    time.Time
	ParsedSummary string
	BlockCount    int
	Materials     []string
}

var tempUploadStore = struct {
	sync.RWMutex
	m map[string]tempUpload
}{
	m: make(map[string]tempUpload),
}

// in-memory counters for schematic views and downloads in tests
var schemCounters = struct {
	sync.RWMutex
	views     map[string]int
	downloads map[string]int
}{
	views:     make(map[string]int),
	downloads: make(map[string]int),
}

// NewTestServer starts a minimal HTTP server exposing endpoints needed for
// HTTP-level regressions without booting the full PocketBase app.
// Currently implements:
//   - GET /           -> 200 OK "ok"
//   - GET /logout     -> clears auth cookie. Normal: 302 to "/". HTMX: 204 + HX-Redirect: "/".
//   - POST /search    -> returns a full HTML doc composed of sidebar + header using templates,
//     auth simulated via presence of the auth cookie.
//   - POST /upload/nbt -> validates .nbt, computes sha256, stores temp, duplicate detection
//   - GET /u/{token}  -> shows stored stats
//   - GET /schematics/{name} -> increments in-memory view counter
//   - GET /download/{name}   -> increments in-memory download counter
//   - GET /_stats/views/{name} and /_stats/downloads/{name} -> return counts as plain text
func NewTestServer(t *testing.T) (baseURL string, cleanup func()) {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
		auth.ClearAuthCookie(w, secure)
		if r.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		// Simulate auth by checking for presence of auth cookie (non-empty)
		isAuth := false
		if c, err := r.Cookie(auth.CookieName); err == nil && strings.TrimSpace(c.Value) != "" {
			isAuth = true
		}
		var header string
		var sidebar string
		if isAuth {
			header = `<div id="header"><a href="/profile">Profile</a> <a href="/logout">Logout</a></div>`
			sidebar = `<div id="sidebar"><a href="/profile">Profile</a> <a href="/settings">Settings</a> <a href="/logout">Logout</a></div>`
		} else {
			header = `<div id="header"><a href="/login">Login</a></div>`
			sidebar = `<div id="sidebar"><a href="/login">Login</a></div>`
		}
		html := "<html><body>" + sidebar + header + `<main id="content">Search Results Placeholder</main>` + "</body></html>"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	mux.HandleFunc("/upload/nbt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// 16MB limit
		_ = r.ParseMultipartForm(16 << 20)
		file, header, err := r.FormFile("nbt")
		if err != nil {
			file, header, err = r.FormFile("file")
			if err != nil {
				http.Error(w, "missing NBT file in form (expected field 'nbt' or 'file')", http.StatusBadRequest)
				return
			}
		}
		defer func() {
			if file != nil {
				_ = file.Close()
			}
		}()
		name := ""
		if header != nil {
			name = header.Filename
		}
		if name == "" || !strings.HasSuffix(strings.ToLower(name), ".nbt") {
			http.Error(w, "invalid file type: expected .nbt", http.StatusBadRequest)
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "failed to read uploaded file", http.StatusInternalServerError)
			return
		}
		// Minimal backend validation to mirror production behavior
		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])
		// duplicate check
		tempUploadStore.RLock()
		for _, v := range tempUploadStore.m {
			if v.Checksum == checksum {
				tempUploadStore.RUnlock()
				http.Error(w, "This schematic already exists (duplicate upload detected by checksum). If you recently uploaded this it may be pending moderation, otherwise it may be blacklisted by the original creator. If you need more help contact us: /contact", http.StatusConflict)
				return
			}
		}
		tempUploadStore.RUnlock()
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			http.Error(w, "failed to generate preview token", http.StatusInternalServerError)
			return
		}
		token := hex.EncodeToString(buf)
		summary, ok := nbtparser.ParseSummary(data)
		parsed := ""
		if ok && summary != "" {
			parsed = summary
		} else {
			parsed = "nbt=unparsed"
		}
		// Extract basic stats (stub)
		bc, mats, _ := nbtparser.ExtractStats(data)
		tempUploadStore.Lock()
		tempUploadStore.m[token] = tempUpload{Filename: name, Size: n, Checksum: checksum, UploadedAt: time.Now(), ParsedSummary: "size=" + formatInt(n) + " bytes; " + parsed, BlockCount: bc, Materials: mats}
		tempUploadStore.Unlock()
		// Return JSON response matching production format
		resp := map[string]interface{}{
			"token":       token,
			"url":         "/u/" + token,
			"checksum":    checksum,
			"filename":    name,
			"size":        n,
			"block_count": bc,
			"dimensions":  map[string]int{"x": 0, "y": 0, "z": 0},
			"materials":   []interface{}{},
			"mods":        []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/u/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/u/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Handle make-public: /u/{token}/make-public
		if strings.HasSuffix(path, "/make-public") {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			rest := strings.TrimPrefix(path, "/u/") // {token}/make-public
			token := strings.TrimSuffix(rest, "/make-public")
			token = strings.TrimSuffix(token, "/") // tolerate trailing slash
			if token == "" {
				http.Error(w, "missing token", http.StatusBadRequest)
				return
			}
			tempUploadStore.RLock()
			_, ok := tempUploadStore.m[token]
			tempUploadStore.RUnlock()
			if !ok {
				http.Error(w, "invalid or expired token", http.StatusNotFound)
				return
			}
			if r.Header.Get("HX-Request") != "" {
				w.Header().Set("HX-Redirect", "/upload/pending")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Redirect(w, r, "/upload/pending", http.StatusSeeOther)
			return
		}
		// Otherwise: preview /u/{token}
		token := strings.TrimPrefix(path, "/u/")
		// strip any stray slash after token
		token = strings.TrimSuffix(token, "/")
		tempUploadStore.RLock()
		v, ok := tempUploadStore.m[token]
		tempUploadStore.RUnlock()
		if !ok {
			http.Error(w, "invalid or expired token", http.StatusNotFound)
			return
		}
		html := "<html><body><h1>Private Preview</h1><ul>" +
			"<li>Filename: " + v.Filename + "</li>" +
			"<li>Size: " + formatInt(v.Size) + " bytes</li>" +
			"<li>SHA-256: " + v.Checksum + "</li>" +
			"<li>Parsed: " + v.ParsedSummary + "</li>" +
			"<li>Block Count: " + formatInt(int64(v.BlockCount)) + "</li>" +
			"<li>Materials: " + materialsToHTML(v.Materials) + "</li>" +
			"</ul></body></html>"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	// schematic view counter endpoint (test-only)
	mux.HandleFunc("/schematics/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/schematics/")
		name = strings.TrimSuffix(name, "/")
		if name == "" {
			http.Error(w, "missing schematic name", http.StatusBadRequest)
			return
		}
		schemCounters.Lock()
		schemCounters.views[name] = schemCounters.views[name] + 1
		schemCounters.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("schematic page: " + name))
	})

	// schematic download counter endpoint (test-only)
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/download/")
		name = strings.TrimSuffix(name, "/")
		if name == "" {
			http.Error(w, "missing schematic name", http.StatusBadRequest)
			return
		}
		schemCounters.Lock()
		schemCounters.downloads[name] = schemCounters.downloads[name] + 1
		schemCounters.Unlock()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("nbt-bytes"))
	})

	// stats endpoints for assertions
	mux.HandleFunc("/_stats/views/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/_stats/views/")
		name = strings.TrimSuffix(name, "/")
		schemCounters.RLock()
		count := schemCounters.views[name]
		schemCounters.RUnlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.Itoa(count)))
	})
	mux.HandleFunc("/_stats/downloads/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/_stats/downloads/")
		name = strings.TrimSuffix(name, "/")
		schemCounters.RLock()
		count := schemCounters.downloads[name]
		schemCounters.RUnlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.Itoa(count)))
	})

	// reports endpoint (test-only) to verify redirect behavior
	mux.HandleFunc("/reports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		ret := r.FormValue("return_to")
		if ret == "" {
			ret = "/"
		}
		typev := r.FormValue("target_type")
		idv := r.FormValue("target_id")
		reason := r.FormValue("reason")
		if typev == "" || idv == "" || reason == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}
		if r.Header.Get("HX-Request") != "" {
			w.Header().Set("HX-Redirect", ret)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, ret, http.StatusSeeOther)
	})

	ts := httptest.NewServer(mux)
	return ts.URL, ts.Close
}

func formatInt(n int64) string {
	return strconv.FormatInt(n, 10)
}

func materialsToHTML(list []string) string {
	if len(list) == 0 {
		return "none"
	}
	return strings.Join(list, ", ")
}
