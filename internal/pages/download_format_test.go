package pages

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"createmod/internal/schematic"
	"createmod/internal/server"
)

func Test_GeneratorDownload_FormatParam(t *testing.T) {
	body := `{"blades":3,"length":8,"rootChord":3,"tipChord":1,"airfoilShape":"linear","bladeMaterial":"wool","bladeColor":"white"}`

	// Default: .nbt
	req := httptest.NewRequest(http.MethodPost, "/api/generators/propeller/download", strings.NewReader(body))
	rec := httptest.NewRecorder()
	if err := GeneratorDownloadHandler("propeller")(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("nbt status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), ".nbt") {
		t.Errorf("disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	nbtBytes := append([]byte(nil), rec.Body.Bytes()...)

	// format=litematic converts the same output
	req = httptest.NewRequest(http.MethodPost, "/api/generators/propeller/download?format=litematic", strings.NewReader(body))
	rec = httptest.NewRecorder()
	if err := GeneratorDownloadHandler("propeller")(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("litematic status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), ".litematic") {
		t.Errorf("disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	lit, err := schematic.ReadLitematic(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("output not a litematic: %v", err)
	}
	src, err := schematic.ReadStructureNBT(nbtBytes)
	if err != nil {
		t.Fatal(err)
	}
	if lit.BlockCount() != src.BlockCount() {
		t.Errorf("block count changed: %d -> %d", src.BlockCount(), lit.BlockCount())
	}

	// Deterministic across calls: same params + format = same bytes
	req = httptest.NewRequest(http.MethodPost, "/api/generators/propeller/download?format=litematic", strings.NewReader(body))
	rec2 := httptest.NewRecorder()
	if err := GeneratorDownloadHandler("propeller")(&server.RequestEvent{Response: rec2, Request: req}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec.Body.Bytes(), rec2.Body.Bytes()) {
		t.Errorf("generator format output is not deterministic")
	}

	// Lossy target carries warnings header
	req = httptest.NewRequest(http.MethodPost, "/api/generators/propeller/download?format=schematic", strings.NewReader(body))
	rec = httptest.NewRecorder()
	if err := GeneratorDownloadHandler("propeller")(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Header().Get("X-Conversion-Warnings") == "" {
		t.Errorf("legacy format download missing warnings header")
	}

	// Unknown format rejected (handler returns a typed APIError that
	// Adapt() maps to 400 in production)
	req = httptest.NewRequest(http.MethodPost, "/api/generators/propeller/download?format=exe", strings.NewReader(body))
	rec = httptest.NewRecorder()
	if err := GeneratorDownloadHandler("propeller")(&server.RequestEvent{Response: rec, Request: req}); err == nil {
		t.Errorf("unknown format accepted")
	}
}

func Test_SchematicDownloadSplit_Model(t *testing.T) {
	d := schematicDownloadSplit("my-build", "en")
	if d.Primary.Href != "/get/my-build" {
		t.Errorf("primary href = %s", d.Primary.Href)
	}
	if len(d.Items) != 4 {
		t.Fatalf("items = %d", len(d.Items))
	}
	wantHrefs := map[string]bool{
		"/get/my-build":                  true,
		"/get/my-build?format=schem":     true,
		"/get/my-build?format=litematic": true,
		"/get/my-build?format=schematic": true,
	}
	lossyCount := 0
	for _, it := range d.Items {
		if !wantHrefs[it.Href] {
			t.Errorf("unexpected href %s", it.Href)
		}
		if it.Lossy {
			lossyCount++
			if !strings.Contains(it.Label, ".schematic") {
				t.Errorf("lossy flag on %s", it.Label)
			}
		}
	}
	if lossyCount != 1 {
		t.Errorf("lossy items = %d", lossyCount)
	}
}

func Test_DownloadSplit_Include_And_Pages(t *testing.T) {
	inc, err := os.ReadFile(filepath.Join("..", "..", "template", "include", "download_split.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{`{{ define "download_split.html" }}`, "data-dl-split", "dl-split-toggle", `role="menu"`, "aria-haspopup"} {
		if !strings.Contains(string(inc), m) {
			t.Errorf("include missing %s", m)
		}
	}

	sch, err := os.ReadFile(filepath.Join("..", "..", "template", "schematic.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sch), `{{ template "download_split.html" .DownloadSplit }}`) {
		t.Errorf("schematic.html does not render the split component")
	}
	if strings.Contains(string(sch), `class="btn btn-primary" id="download-btn"`) {
		t.Errorf("old download anchor still present")
	}

	for _, tpl := range []string{"generator-propeller.html", "generator-balloon.html", "generator-hull.html"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "template", tpl))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, m := range []string{"data-dl-split", `data-gen-format="schem"`, `data-gen-format="litematic"`, `data-gen-format="schematic"`, "?format=' + this.getAttribute('data-gen-format')"} {
			if !strings.Contains(s, m) {
				t.Errorf("%s missing %s", tpl, m)
			}
		}
	}

	foot, err := os.ReadFile(filepath.Join("..", "..", "template", "include", "foot.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(foot), "download-split.js") {
		t.Errorf("foot.html does not load download-split.js")
	}
}

func Test_ConvCacheKey_ChangesOnUpdate(t *testing.T) {
	s1 := convCacheKey("abc", mustTime(t, "2026-07-01T10:00:00Z"), "build", ".schem")
	s2 := convCacheKey("abc", mustTime(t, "2026-07-02T10:00:00Z"), "build", ".schem")
	if s1 == s2 {
		t.Errorf("cache key must change when Updated changes")
	}
	if !strings.HasPrefix(s1, "_conv/v1/abc/") {
		t.Errorf("key = %s", s1)
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
