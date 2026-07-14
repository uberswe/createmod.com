package pages

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		"h1 class=\"h2 mb-1\"",
		"/api/convert/batch",
		"multiple",
		"conv-filelist",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("convert.html missing: %s", m)
		}
	}
}

func Test_ConvertPairs_AllOrderedPairs(t *testing.T) {
	pairs := ConvertPairs()
	if len(pairs) != 30 {
		t.Fatalf("expected 30 ordered pairs (6 formats), got %d", len(pairs))
	}
	seen := map[string]bool{}
	for _, p := range pairs {
		if p.FromSlug == p.ToSlug {
			t.Errorf("self pair %s", p.FromSlug)
		}
		seen[p.FromSlug+"-to-"+p.ToSlug] = true
	}
	for _, want := range []string{"nbt-to-schem", "nbt-to-litematic", "schem-to-nbt", "schem-to-litematic", "litematic-to-nbt", "litematic-to-schem", "schematic-to-nbt", "nbt-to-schematic", "schematic-to-litematic", "litematic-to-schematic", "schematic-to-schem", "schem-to-schematic"} {
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

// multipartBatchBody builds a multipart body with repeated "files" parts.
func multipartBatchBody(t *testing.T, to string, files map[string][]byte, names []string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("to", to); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		fw, err := w.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, w.FormDataContentType()
}

func runBatch(t *testing.T, body *bytes.Buffer, ctype string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/convert/batch", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	if err := ConvertBatchAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	return rec
}

func Test_ConvertBatchAPIHandler_ConvertsMultipleToZip(t *testing.T) {
	nbt := testStructureNBT(t)
	body, ctype := multipartBatchBody(t, "schem", map[string][]byte{"one.nbt": nbt, "two.nbt": nbt}, []string{"one.nbt", "two.nbt"})
	rec := runBatch(t, body, ctype)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("content type = %q", ct)
	}
	if dispo := rec.Header().Get("Content-Disposition"); !strings.Contains(dispo, "converted-schem.zip") {
		t.Errorf("content disposition = %q", dispo)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("output is not a zip: %v", err)
	}
	got := map[string]bool{}
	for _, f := range zr.File {
		got[f.Name] = true
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out, err := schematic.ReadSponge(data)
		if err != nil {
			t.Fatalf("%s unreadable: %v", f.Name, err)
		}
		if out.BlockCount() != 1 {
			t.Errorf("%s content lost: blocks=%d", f.Name, out.BlockCount())
		}
	}
	if !got["one.schem"] || !got["two.schem"] {
		t.Errorf("zip entries = %v", got)
	}
	var results []convertBatchResult
	if err := json.Unmarshal([]byte(rec.Header().Get("X-Conversion-Results")), &results); err != nil {
		t.Fatalf("results header: %v", err)
	}
	if len(results) != 2 || !results[0].OK || !results[1].OK {
		t.Errorf("results = %+v", results)
	}
}

func Test_ConvertBatchAPIHandler_DedupesOutputNames(t *testing.T) {
	nbt := testStructureNBT(t)
	// Same base name from different folders collides after conversion.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("to", "schem"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		fw, err := w.CreateFormFile("files", "build.nbt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(nbt); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	rec := runBatch(t, &buf, w.FormDataContentType())
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		if names[f.Name] {
			t.Fatalf("duplicate zip entry %s", f.Name)
		}
		names[f.Name] = true
	}
	if len(names) != 2 {
		t.Errorf("expected 2 distinct entries, got %v", names)
	}
}

func Test_ConvertBatchAPIHandler_PartialFailure(t *testing.T) {
	body, ctype := multipartBatchBody(t, "schem",
		map[string][]byte{"good.nbt": testStructureNBT(t), "bad.nbt": []byte("not a schematic")},
		[]string{"good.nbt", "bad.nbt"})
	rec := runBatch(t, body, ctype)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var results []convertBatchResult
	if err := json.Unmarshal([]byte(rec.Header().Get("X-Conversion-Results")), &results); err != nil {
		t.Fatalf("results header: %v", err)
	}
	if len(results) != 2 || !results[0].OK || results[1].OK || results[1].Error == "" {
		t.Errorf("results = %+v", results)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != 1 {
		t.Errorf("expected 1 zip entry, got %d", len(zr.File))
	}
}

func Test_ConvertBatchAPIHandler_RejectsBadBatches(t *testing.T) {
	// Every file fails → 422 with per-file results.
	body, ctype := multipartBatchBody(t, "schem",
		map[string][]byte{"a.nbt": []byte("junk"), "b.nbt": []byte("junk")},
		[]string{"a.nbt", "b.nbt"})
	rec := runBatch(t, body, ctype)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("all-fail: status %d", rec.Code)
	}

	// Unsupported target format.
	body, ctype = multipartBatchBody(t, "exe", map[string][]byte{"a.nbt": testStructureNBT(t)}, []string{"a.nbt"})
	rec = runBatch(t, body, ctype)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad target: status %d", rec.Code)
	}

	// No files at all.
	body, ctype = multipartBatchBody(t, "schem", nil, nil)
	rec = runBatch(t, body, ctype)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("no files: status %d", rec.Code)
	}

	// Too many files.
	nbt := testStructureNBT(t)
	files := map[string][]byte{}
	var names []string
	for i := 0; i < maxConvertBatchFiles+1; i++ {
		name := fmt.Sprintf("f%d.nbt", i)
		files[name] = nbt
		names = append(names, name)
	}
	body, ctype = multipartBatchBody(t, "schem", files, names)
	rec = runBatch(t, body, ctype)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("too many: status %d", rec.Code)
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
