package pages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/schematic"
	"createmod/internal/store"
)

func Test_EditorSourceModel_Blank(t *testing.T) {
	sess := &store.EditorSession{SourceKind: "blank", SourceRef: "[8,4,6]"}
	m, err := editorSourceModel(context.Background(), nil, nil, sess)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != [3]int{8, 4, 6} || m.DataVersion != 3955 {
		t.Errorf("blank model: %v dv=%d", m.Size, m.DataVersion)
	}
	// out-of-range dims rejected
	for _, bad := range []string{"[0,4,6]", "[8,4,999]", "notjson"} {
		sess.SourceRef = bad
		if _, err := editorSourceModel(context.Background(), nil, nil, sess); err == nil {
			t.Errorf("blank %q accepted", bad)
		}
	}
}

func Test_EditorCurrentModel_Replay(t *testing.T) {
	ops := []schematic.Op{
		{Type: "fill", Min: [3]int{0, 0, 0}, Max: [3]int{3, 0, 3}, Block: "minecraft:stone"},
		{Type: "fill", Min: [3]int{1, 1, 1}, Max: [3]int{2, 2, 2}, Block: "minecraft:oak_planks"},
		{Type: "rotate", Steps: 1},
	}
	opsJSON, _ := json.Marshal(ops)

	// cursor at 2: only the two fills applied
	sess := &store.EditorSession{SourceKind: "blank", SourceRef: "[4,3,4]", Ops: opsJSON, Cursor: 2}
	m, gotOps, err := editorCurrentModel(context.Background(), nil, nil, sess)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotOps) != 3 {
		t.Errorf("ops = %d", len(gotOps))
	}
	if m.BlockCount() != 16+8 {
		t.Errorf("blocks = %d, want 24", m.BlockCount())
	}
	// cursor at 3: rotation applied too (square footprint so same size)
	sess.Cursor = 3
	m3, _, err := editorCurrentModel(context.Background(), nil, nil, sess)
	if err != nil {
		t.Fatal(err)
	}
	if m3.BlockCount() != 24 {
		t.Errorf("after rotate blocks = %d", m3.BlockCount())
	}
	// corrupt cursor rejected
	sess.Cursor = 99
	if _, _, err := editorCurrentModel(context.Background(), nil, nil, sess); err == nil {
		t.Errorf("bad cursor accepted")
	}
}

func Test_EditorPreviewTypeMapping(t *testing.T) {
	if previewTypeFor(schematic.BlockState{Name: "minecraft:oak_stairs"}) != 2 {
		t.Errorf("stairs type")
	}
	if previewTypeFor(schematic.BlockState{Name: "minecraft:stone_slab"}) != 3 {
		t.Errorf("slab type")
	}
	if previewTypeFor(schematic.BlockState{Name: "create:cogwheel"}) != 1 {
		t.Errorf("cube fallback")
	}
	props := previewPropsFor(schematic.BlockState{Name: "minecraft:oak_stairs", Properties: map[string]string{"facing": "east", "half": "top"}})
	if props["facing"] != "east" || props["half"] != "top" {
		t.Errorf("stair props: %v", props)
	}
}

func Test_Editor_Templates(t *testing.T) {
	ed, err := os.ReadFile(filepath.Join("..", "..", "template", "editor.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{
		"ed-drop", "editor-canvas", "/api/editor/sessions", "'/op'", "/undo", "/redo",
		"preview.nbt", "preview.json", "/upload/nbt", "GeneratorApp.initScene",
		"data-op=\"crop\"", "data-op=\"rotate\"", "data-op=\"mirror-x\"", "data-op=\"replace\"",
	} {
		if !strings.Contains(string(ed), m) {
			t.Errorf("editor.html missing %q", m)
		}
	}
	sch, err := os.ReadFile(filepath.Join("..", "..", "template", "schematic.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sch), "/tools/editor?source={{ .Schematic.Name }}") {
		t.Errorf("schematic.html missing edit entry point")
	}
	hdr, err := os.ReadFile(filepath.Join("..", "..", "template", "include", "header.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hdr), "header-editor-btn") || !strings.Contains(string(hdr), "/tools/editor") {
		t.Errorf("header missing editor entry")
	}
}
