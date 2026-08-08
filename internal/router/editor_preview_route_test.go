package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// The plain preview.nbt route and the token-in-path variant must coexist: a
// 2-segment request hits the plain route, a 3-segment request hits the path
// variant and exposes both {id} and {token}. This guards the external-viewer
// fix against a chi routing ambiguity.
func TestEditorPreviewRoutes_Coexist(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/editor/{id}/preview.nbt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Route", "plain")
		w.Header().Set("X-Id", chi.URLParam(req, "id"))
	})
	r.Get("/api/editor/{id}/{token}/preview.nbt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Route", "path-token")
		w.Header().Set("X-Id", chi.URLParam(req, "id"))
		w.Header().Set("X-Token", chi.URLParam(req, "token"))
	})

	do := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	plain := do("/api/editor/sess-1/preview.nbt")
	if plain.Header().Get("X-Route") != "plain" || plain.Header().Get("X-Id") != "sess-1" {
		t.Fatalf("plain route: got route=%q id=%q", plain.Header().Get("X-Route"), plain.Header().Get("X-Id"))
	}

	// Token segment carries dots (scope.exp.sig); it must be captured whole.
	pathTok := do("/api/editor/sess-1/v.68abc123.deadbeefdeadbeef/preview.nbt")
	if pathTok.Header().Get("X-Route") != "path-token" {
		t.Fatalf("path-token route not matched: got %q", pathTok.Header().Get("X-Route"))
	}
	if pathTok.Header().Get("X-Id") != "sess-1" {
		t.Fatalf("path-token id: got %q, want sess-1", pathTok.Header().Get("X-Id"))
	}
	if got := pathTok.Header().Get("X-Token"); got != "v.68abc123.deadbeefdeadbeef" {
		t.Fatalf("path-token token: got %q", got)
	}
}
