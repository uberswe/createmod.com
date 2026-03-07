package server

import (
	"bytes"
	"path/filepath"
	"html/template"
	"strings"
	"sync"
)

// Registry is a drop-in replacement for pocketbase/tools/template.Registry.
// It provides thread-safe template caching, custom function registration,
// and the same LoadFiles(...).Render(data) API.
type Registry struct {
	mu    sync.RWMutex
	cache map[string]*Renderer
	funcs template.FuncMap
}

// NewRegistry creates a new template registry with a built-in "raw" helper
// that outputs unescaped HTML (matching PocketBase's default).
func NewRegistry() *Registry {
	return &Registry{
		cache: make(map[string]*Renderer),
		funcs: template.FuncMap{
			"raw": func(s string) template.HTML { return template.HTML(s) },
		},
	}
}

// AddFuncs registers additional template functions. Existing names are replaced.
func (r *Registry) AddFuncs(funcs template.FuncMap) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range funcs {
		r.funcs[k] = v
	}
	return r
}

// LoadFiles parses the given template files and returns a Renderer.
// Results are cached by the joined filenames; subsequent calls with the same
// files return the cached Renderer.
func (r *Registry) LoadFiles(filenames ...string) *Renderer {
	key := strings.Join(filenames, ",")

	r.mu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		return cached
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if cached, ok := r.cache[key]; ok {
		return cached
	}

	t, err := template.New("").Funcs(r.funcs).ParseFiles(filenames...)
	name := ""
	if len(filenames) > 0 {
		name = filepath.Base(filenames[0])
	}
	renderer := &Renderer{template: t, name: name, parseError: err}
	r.cache[key] = renderer
	return renderer
}

// Renderer holds a parsed template and renders it with data.
type Renderer struct {
	template   *template.Template
	name       string
	parseError error
}

// Render executes the parsed template with the given data and returns the
// resulting HTML string.
func (r *Renderer) Render(data any) (string, error) {
	if r.parseError != nil {
		return "", r.parseError
	}
	var buf bytes.Buffer
	if err := r.template.ExecuteTemplate(&buf, r.name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
