package pages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/schematic"
	"createmod/internal/server"

	"github.com/Tnze/go-mc/nbt"
)

// suspiciousNBT builds a structure file with a command block.
func suspiciousNBT(t *testing.T) []byte {
	t.Helper()
	s := schematic.New(2, 1, 1)
	s.DataVersion = 3955
	cb := s.PaletteIndex(schematic.BlockState{Name: "minecraft:command_block"})
	s.Blocks[0] = cb

	var buf strings.Builder
	_ = buf
	beFields := map[string]nbt.RawMessage{}
	// command via SNBT-free construction: TAG_String payload
	cmd := "/give @p diamond"
	payload := make([]byte, 2+len(cmd))
	payload[0] = byte(len(cmd) >> 8)
	payload[1] = byte(len(cmd))
	copy(payload[2:], cmd)
	beFields["Command"] = nbt.RawMessage{Type: nbt.TagString, Data: payload}
	id := "minecraft:command_block"
	idPayload := make([]byte, 2+len(id))
	idPayload[0] = byte(len(id) >> 8)
	idPayload[1] = byte(len(id))
	copy(idPayload[2:], id)
	beFields["id"] = nbt.RawMessage{Type: nbt.TagString, Data: idPayload}

	// Build compound payload manually: sorted fields via schematic helper is
	// unexported; emit through a struct instead.
	type cbBE struct {
		ID      string `nbt:"id"`
		Command string `nbt:"Command"`
	}
	var nbtBuf strings.Builder
	_ = nbtBuf
	var raw []byte
	{
		var b []byte
		bw := &sliceWriter{b: &b}
		if err := nbt.NewEncoder(bw).Encode(cbBE{ID: id, Command: cmd}, ""); err != nil {
			t.Fatal(err)
		}
		raw = b[3:] // strip root header
	}
	s.BlockEntities = append(s.BlockEntities, schematic.BlockEntity{
		Pos: [3]int{0, 0, 0},
		Raw: nbt.RawMessage{Type: nbt.TagCompound, Data: raw},
	})
	data, err := schematic.WriteStructureNBT(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type sliceWriter struct{ b *[]byte }

func (w *sliceWriter) Write(p []byte) (int, error) {
	*w.b = append(*w.b, p...)
	return len(p), nil
}

func Test_SafetyCheckAPI(t *testing.T) {
	// Suspicious file: valid, notable, command extracted
	body, ctype := multipartBody(t, nil, "file", "sus.nbt", suspiciousNBT(t))
	req := httptest.NewRequest(http.MethodPost, "/api/safety-check", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	if err := SafetyCheckAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["fileSafe"] != true || resp["notable"] != true {
		t.Errorf("resp: fileSafe=%v notable=%v", resp["fileSafe"], resp["notable"])
	}
	if s, _ := resp["summary"].(string); !strings.Contains(s, "command block") {
		t.Errorf("summary = %v", resp["summary"])
	}

	// Clean file: validated
	body, ctype = multipartBody(t, nil, "file", "ok.nbt", testStructureNBT(t))
	req = httptest.NewRequest(http.MethodPost, "/api/safety-check", body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := SafetyCheckAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	resp = map[string]interface{}{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["fileSafe"] != true || resp["notable"] != false {
		t.Errorf("clean: fileSafe=%v notable=%v", resp["fileSafe"], resp["notable"])
	}

	// Garbage: fileSafe=false with parse error, still 200 (a result, not an error)
	body, ctype = multipartBody(t, nil, "file", "junk.nbt", []byte("junk"))
	req = httptest.NewRequest(http.MethodPost, "/api/safety-check", body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := SafetyCheckAPIHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	resp = map[string]interface{}{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != http.StatusOK || resp["fileSafe"] != false || resp["parseError"] == nil {
		t.Errorf("garbage: code=%d resp=%v", rec.Code, resp)
	}
}

func Test_Safety_Templates(t *testing.T) {
	// The explainer content lives on the checker page (/safety redirects there).
	checker, err := os.ReadFile(filepath.Join("..", "..", "template", "safety_check.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{
		"sc-drop", "/api/safety-check", "never stored",
		"safety-explainer", "schematics are data, not programs", "FAQPage",
	} {
		if !strings.Contains(strings.ToLower(string(checker)), strings.ToLower(m)) {
			t.Errorf("safety_check.html missing %q", m)
		}
	}
	sch, err := os.ReadFile(filepath.Join("..", "..", "template", "schematic.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{".Safety.Scanned", ".Safety.FileSafe", ".Safety.Summary", "Validated"} {
		if !strings.Contains(string(sch), m) {
			t.Errorf("schematic.html missing %q", m)
		}
	}
}

func Test_ManifestSummary(t *testing.T) {
	m := &schematic.Manifest{Counts: map[schematic.FindingType]int{
		schematic.FindingCommandBlock: 2,
		schematic.FindingSpawner:      1,
	}}
	got := manifestSummary(m)
	if got != "2 command block(s), 1 mob spawner(s)" {
		t.Errorf("summary = %q", got)
	}
}
