package schematic

import (
	"encoding/json"
	"strings"
	"testing"
)

func Test_Blueprint_RoundTrip(t *testing.T) {
	for name, data := range map[string][]byte{
		"generator": generatorFixture(t),
		"handmade":  handmadeFixture(t),
	} {
		src, err := ReadStructureNBT(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		src.Meta.Name = "test build"
		bp, err := WriteBlueprint(src)
		if err != nil {
			t.Fatalf("%s: write blueprint: %v", name, err)
		}
		if f, err := Detect(bp); err != nil || f != FormatBlueprint {
			t.Fatalf("%s: detect = %v %v", name, f, err)
		}
		back, err := ReadBlueprint(bp)
		if err != nil {
			t.Fatalf("%s: read blueprint: %v", name, err)
		}
		if src.Size != back.Size {
			t.Errorf("%s: size %v -> %v", name, src.Size, back.Size)
		}
		if src.DataVersion != back.DataVersion {
			t.Errorf("%s: DataVersion %d -> %d (mcversion)", name, src.DataVersion, back.DataVersion)
		}
		if back.Meta.Name != "test build" {
			t.Errorf("%s: name = %q", name, back.Meta.Name)
		}
		if len(src.BlockEntities) != len(back.BlockEntities) {
			t.Errorf("%s: block entities %d -> %d", name, len(src.BlockEntities), len(back.BlockEntities))
		}
		for y := 0; y < src.Size[1]; y++ {
			for z := 0; z < src.Size[2]; z++ {
				for x := 0; x < src.Size[0]; x++ {
					if src.At(x, y, z).Key() != back.At(x, y, z).Key() {
						t.Fatalf("%s: block at (%d,%d,%d) changed", name, x, y, z)
					}
				}
			}
		}
	}
}

func Test_Blueprint_OddVolumePacking(t *testing.T) {
	// 3x1x1 = odd volume exercises the padded final int.
	s := New(3, 1, 1)
	s.DataVersion = 3955
	stone := s.PaletteIndex(BlockState{Name: "minecraft:stone"})
	s.Blocks[0] = stone
	s.Blocks[2] = stone
	bp, err := WriteBlueprint(s)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ReadBlueprint(bp)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"minecraft:stone", "minecraft:air", "minecraft:stone"} {
		if got := back.Palette[back.Blocks[i]].Name; got != want {
			t.Errorf("block %d = %s, want %s", i, got, want)
		}
	}
}

func Test_BG2_RoundTrip(t *testing.T) {
	src, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	src.Meta.Name = "hull"
	out, warnings, err := WriteBuildingGadgets(src)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings for BE-free build: %v", warnings)
	}
	// Must be valid JSON with the BG2 shape
	var doc map[string]interface{}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	if doc["statePosArrayList"] == "" || doc["name"] != "hull" {
		t.Errorf("doc shape: name=%v", doc["name"])
	}
	if f, err := Detect(out); err != nil || f != FormatBG {
		t.Fatalf("detect = %v %v", f, err)
	}
	back, err := ReadBuildingGadgets(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if src.Size != back.Size {
		t.Errorf("size %v -> %v", src.Size, back.Size)
	}
	if src.BlockCount() != back.BlockCount() {
		t.Errorf("blocks %d -> %d", src.BlockCount(), back.BlockCount())
	}
	for y := 0; y < src.Size[1]; y++ {
		for z := 0; z < src.Size[2]; z++ {
			for x := 0; x < src.Size[0]; x++ {
				if src.At(x, y, z).Key() != back.At(x, y, z).Key() {
					t.Fatalf("block at (%d,%d,%d) changed", x, y, z)
				}
			}
		}
	}
}

func Test_BG2_DropsBlockEntitiesWithWarning(t *testing.T) {
	src, err := ReadStructureNBT(handmadeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	_, warnings, err := WriteBuildingGadgets(src)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "block entit") {
			found = true
		}
	}
	if !found {
		t.Errorf("no block-entity warning: %v", warnings)
	}
}

func Test_BG1_Read(t *testing.T) {
	// Hand-build a v1 paste string: two blocks at (10,64,10) and (11,64,10).
	type stateEntry struct {
		State structPaletteEntryOut `nbt:"state"`
	}
	packLong := func(state, x, y, z int) int64 {
		return int64(state)<<40 | int64(x)<<24 | int64(y)<<16 | int64(z)
	}
	body := struct {
		Pos  []int64      `nbt:"pos"`
		Data []stateEntry `nbt:"data"`
	}{
		Pos: []int64{
			packLong(0, 10, 64, 10),
			packLong(1, 11, 64, 10),
		},
		Data: []stateEntry{
			{State: structPaletteEntryOut{Name: "minecraft:stone"}},
			{State: structPaletteEntryOut{Name: "minecraft:oak_planks"}},
		},
	}
	nbtBytes, err := encodeAndGzip(body)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]interface{}{
		"header": map[string]interface{}{
			"bounding_box": map[string]int{"min_x": 10, "min_y": 64, "min_z": 10, "max_x": 11, "max_y": 64, "max_z": 10},
		},
		"body": base64Encode(nbtBytes),
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}

	if f, err := Detect(data); err != nil || f != FormatBG {
		t.Fatalf("detect = %v %v", f, err)
	}
	s, err := ReadBuildingGadgets(data)
	if err != nil {
		t.Fatalf("read v1: %v", err)
	}
	if s.Size != [3]int{2, 1, 1} {
		t.Fatalf("size = %v", s.Size)
	}
	if s.At(0, 0, 0).Name != "minecraft:stone" || s.At(1, 0, 0).Name != "minecraft:oak_planks" {
		t.Errorf("blocks: %s, %s", s.At(0, 0, 0).Name, s.At(1, 0, 0).Name)
	}
}

func Test_NewFormats_ViaConvertMatrix(t *testing.T) {
	src, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []Format{FormatBlueprint, FormatBG} {
		out, err := Write(src, target)
		if err != nil {
			t.Fatalf("write %s: %v", target, err)
		}
		res, err := Convert(out, FormatStructure)
		if err != nil {
			t.Fatalf("%s -> structure: %v", target, err)
		}
		if res.From != target {
			t.Errorf("detected %s as %s", target, res.From)
		}
		back, err := ReadStructureNBT(res.Data)
		if err != nil {
			t.Fatal(err)
		}
		if back.BlockCount() != src.BlockCount() {
			t.Errorf("%s: blocks %d -> %d", target, src.BlockCount(), back.BlockCount())
		}
	}
}
