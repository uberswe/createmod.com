package pages

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/schematic"
	"createmod/internal/server"
)

func Test_Convert_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "convert.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"conv-drop",
		"conv-file",
		"conv-targets",
		"/api/convert/inspect",
		"/api/convert",
		"data-to=\"nbt\"",
		"data-to=\"schem\"",
		"data-to=\"litematic\"",
		"SoftwareApplication",
		"h1 class=\"h2 page-title\"",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("convert.html missing: %s", m)
		}
	}
}

func Test_ConvertPairs_AllOrderedPairs(t *testing.T) {
	pairs := ConvertPairs()
	if len(pairs) != 6 {
		t.Fatalf("expected 6 ordered pairs, got %d", len(pairs))
	}
	seen := map[string]bool{}
	for _, p := range pairs {
		if p.FromSlug == p.ToSlug {
			t.Errorf("self pair %s", p.FromSlug)
		}
		seen[p.FromSlug+"-to-"+p.ToSlug] = true
	}
	for _, want := range []string{"nbt-to-schem", "nbt-to-litematic", "schem-to-nbt", "schem-to-litematic", "litematic-to-nbt", "litematic-to-schem"} {
		if !seen[want] {
			t.Errorf("missing pair %s", want)
		}
	}
}

func multipartBody(t *testing.T, fields map[string]string, fileField, fileName string, fileData []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	fw, err := w.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(fileData); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

// testStructureNBT builds a tiny valid structure NBT via the schematic package.
func testStructureNBT(t *testing.T) []byte {
	t.Helper()
	s := schematic.New(2, 1, 1)
	s.DataVersion = 3955
	idx := s.PaletteIndex(schematic.BlockState{Name: "minecraft:stone"})
	s.Blocks[0] = idx
	data, err := schematic.WriteStructureNBT(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func Test_ConvertAPIHandler_ConvertsAndWarnsViaHeaders(t *testing.T) {
	body, ctype := multipartBody(t, map[string]string{"to": "schem"}, "file", "build.nbt", testStructureNBT(t))
	req := httptest.NewRequest(http.MethodPost, "/api/convert", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	e := &server.RequestEvent{Response: rec, Request: req}

	if err := ConvertAPIHandler()(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Detected-Format"); got != "nbt" {
		t.Errorf("detected format header = %q", got)
	}
	if dispo := rec.Header().Get("Content-Disposition"); !strings.Contains(dispo, "build.schem") {
		t.Errorf("content disposition = %q", dispo)
	}
	// Output must be a readable .schem preserving the content.
	out, err := schematic.ReadSponge(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("output unreadable: %v", err)
	}
	if out.BlockCount() != 1 || out.DataVersion != 3955 {
		t.Errorf("content lost: blocks=%d dv=%d", out.BlockCount(), out.DataVersion)
	}
}

func Test_ConvertAPIHandler_RejectsBadInput(t *testing.T) {
	// Unsupported target
	body, ctype := multipartBody(t, map[string]string{"to": "exe"}, "file", "a.nbt", testStructureNBT(t))
	req := httptest.NewRequest(http.MethodPost, "/api/convert", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	if err := ConvertAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad target: status %d", rec.Code)
	}

	// Garbage file
	body, ctype = multipartBody(t, map[string]string{"to": "schem"}, "file", "a.nbt", []byte("not a schematic"))
	req = httptest.NewRequest(http.MethodPost, "/api/convert", body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := ConvertAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("garbage: status %d", rec.Code)
	}
}

func Test_ConvertInspectHandler_IdentifiesFormat(t *testing.T) {
	body, ctype := multipartBody(t, nil, "file", "b.nbt", testStructureNBT(t))
	req := httptest.NewRequest(http.MethodPost, "/api/convert/inspect", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	if err := ConvertInspectHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["format"] != "nbt" {
		t.Errorf("format = %v", resp["format"])
	}
	if bc, ok := resp["blockCount"].(float64); !ok || bc != 1 {
		t.Errorf("blockCount = %v", resp["blockCount"])
	}
}
