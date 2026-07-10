package schematic

import (
	"bytes"
	"reflect"
	"testing"

	"createmod/internal/generator"
	"createmod/internal/nbtparser"

	"github.com/Tnze/go-mc/nbt"
)

// generatorFixture produces a known-good Create-compatible structure NBT via
// the existing generator export path (historically validated in-game).
func generatorFixture(t *testing.T) []byte {
	t.Helper()
	res, err := generator.GenerateHull(generator.HullParams{
		WoodType: "spruce", Length: 24, Beam: 8, Depth: 5,
		BowLength: 6, SternLength: 4, BowSharpness: 1.3, SternSharpness: 0.7,
		SternStyle: "round", KeelCurve: 1.7, CastleBlend: 4, BottomPinch: 0.3,
	})
	if err != nil {
		t.Fatalf("generate fixture: %v", err)
	}
	data, err := generator.ExportNBT(res)
	if err != nil {
		t.Fatalf("export fixture: %v", err)
	}
	return data
}

// handmadeFixture builds a small structure with a block entity, an entity,
// properties, and an explicit DataVersion.
func handmadeFixture(t *testing.T) []byte {
	t.Helper()
	s := New(2, 2, 2)
	s.DataVersion = 3955
	stone := s.PaletteIndex(BlockState{Name: "minecraft:stone"})
	stairs := s.PaletteIndex(BlockState{Name: "minecraft:oak_stairs", Properties: map[string]string{"facing": "east", "half": "bottom"}})
	chest := s.PaletteIndex(BlockState{Name: "minecraft:chest", Properties: map[string]string{"facing": "north"}})
	s.Blocks[s.Index(0, 0, 0)] = stone
	s.Blocks[s.Index(1, 0, 0)] = stairs
	s.Blocks[s.Index(0, 1, 0)] = chest

	var beBuf bytes.Buffer
	type chestNBT struct {
		ID   string `nbt:"id"`
		Lock string `nbt:"Lock"`
	}
	if err := nbt.NewEncoder(&beBuf).Encode(chestNBT{ID: "minecraft:chest", Lock: ""}, ""); err != nil {
		t.Fatal(err)
	}
	s.BlockEntities = append(s.BlockEntities, BlockEntity{
		Pos: [3]int{0, 1, 0},
		Raw: nbt.RawMessage{Type: nbt.TagCompound, Data: beBuf.Bytes()[3:]}, // strip root tag header (type byte + empty name)
	})

	data, err := WriteStructureNBT(s)
	if err != nil {
		t.Fatalf("write handmade fixture: %v", err)
	}
	return data
}

func Test_ReadStructureNBT_GeneratorOutput(t *testing.T) {
	data := generatorFixture(t)
	s, err := ReadStructureNBT(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if s.DataVersion == 0 {
		t.Errorf("DataVersion not read (generator writes 3955)")
	}
	if s.BlockCount() == 0 {
		t.Errorf("no blocks parsed")
	}
	if s.Meta.SourceFormat != "structure" {
		t.Errorf("source format = %q", s.Meta.SourceFormat)
	}

	// Materials must agree with the legacy parser on the same bytes.
	legacy, err := nbtparser.ExtractMaterials(data)
	if err != nil {
		t.Fatalf("legacy materials: %v", err)
	}
	legacyTotal := 0
	for _, m := range legacy {
		legacyTotal += m.Count
	}
	ours := s.Materials()
	ourTotal := 0
	for _, m := range ours {
		ourTotal += m.Count
	}
	if legacyTotal != ourTotal {
		t.Errorf("material totals disagree: legacy=%d ours=%d", legacyTotal, ourTotal)
	}
}

func Test_RoundTrip_ModelEquality(t *testing.T) {
	for name, data := range map[string][]byte{
		"generator": generatorFixture(t),
		"handmade":  handmadeFixture(t),
	} {
		s1, err := ReadStructureNBT(data)
		if err != nil {
			t.Fatalf("%s read 1: %v", name, err)
		}
		out, err := WriteStructureNBT(s1)
		if err != nil {
			t.Fatalf("%s write: %v", name, err)
		}
		s2, err := ReadStructureNBT(out)
		if err != nil {
			t.Fatalf("%s read 2: %v", name, err)
		}
		if !reflect.DeepEqual(s1.Size, s2.Size) {
			t.Errorf("%s: size changed %v -> %v", name, s1.Size, s2.Size)
		}
		if s1.DataVersion != s2.DataVersion {
			t.Errorf("%s: DataVersion changed %d -> %d", name, s1.DataVersion, s2.DataVersion)
		}
		if s1.BlockCount() != s2.BlockCount() {
			t.Errorf("%s: block count changed %d -> %d", name, s1.BlockCount(), s2.BlockCount())
		}
		if !reflect.DeepEqual(s1.Materials(), s2.Materials()) {
			t.Errorf("%s: materials changed", name)
		}
		if len(s1.BlockEntities) != len(s2.BlockEntities) {
			t.Errorf("%s: block entities %d -> %d", name, len(s1.BlockEntities), len(s2.BlockEntities))
		}
		// Every position must carry the same state through the round trip.
		for y := 0; y < s1.Size[1]; y++ {
			for z := 0; z < s1.Size[2]; z++ {
				for x := 0; x < s1.Size[0]; x++ {
					a, b := s1.At(x, y, z), s2.At(x, y, z)
					if a.Key() != b.Key() {
						t.Fatalf("%s: block at (%d,%d,%d) changed %s -> %s", name, x, y, z, a.Key(), b.Key())
					}
				}
			}
		}
	}
}

func Test_WriteStructureNBT_ByteStable(t *testing.T) {
	s, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	out1, err := WriteStructureNBT(s)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := WriteStructureNBT(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out1, out2) {
		t.Errorf("writer output is not deterministic")
	}
}

func Test_WriteStructureNBT_LegacyParserCompat(t *testing.T) {
	// Output must be accepted by the legacy validation path that gates
	// uploads today (which also exercises the TAG_List(Int) requirement).
	s, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := WriteStructureNBT(s)
	if err != nil {
		t.Fatal(err)
	}
	if ok, reason := nbtparser.Validate(out); !ok {
		t.Fatalf("legacy parser rejects our output: %s", reason)
	}
	// The TAG_List(Int) framing is what Create requires; assert the raw
	// bytes do NOT contain the size field as TAG_Int_Array.
	rawOut, err := decompress(out)
	if err != nil {
		t.Fatal(err)
	}
	// TAG_Int_Array (0x0b) immediately before the name "size" would mean the
	// encoder regression returned.
	if bytes.Contains(rawOut, append([]byte{0x0b, 0x00, 0x04}, []byte("size")...)) {
		t.Errorf("size written as TAG_Int_Array; Create requires TAG_List(Int)")
	}
	if !bytes.Contains(rawOut, append([]byte{0x09, 0x00, 0x04}, []byte("size")...)) {
		t.Errorf("size not written as TAG_List")
	}
}

func Test_ReadStructureNBT_Hardening(t *testing.T) {
	// Oversized dimensions must be rejected before allocation.
	var buf bytes.Buffer
	type root struct {
		DataVersion int32   `nbt:"DataVersion"`
		Size        intList `nbt:"size"`
	}
	if err := nbt.NewEncoder(&buf).Encode(root{DataVersion: 3955, Size: intList{30000, 30000, 30000}}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadStructureNBT(buf.Bytes()); err == nil {
		t.Errorf("volume bomb accepted")
	}
	// Garbage input must error, not panic.
	if _, err := ReadStructureNBT([]byte("not nbt at all")); err == nil {
		t.Errorf("garbage accepted")
	}
	if _, err := ReadStructureNBT([]byte{0x1f, 0x8b, 0xff, 0xff}); err == nil {
		t.Errorf("truncated gzip accepted")
	}
}

func FuzzReadStructureNBT(f *testing.F) {
	f.Add(generatorFixture(&testing.T{}))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := ReadStructureNBT(data)
		if err != nil {
			return
		}
		// Anything accepted must be internally consistent and re-writable.
		if err := s.Validate(); err != nil {
			t.Fatalf("accepted schematic fails validation: %v", err)
		}
		if _, err := WriteStructureNBT(s); err != nil {
			t.Fatalf("accepted schematic fails to write: %v", err)
		}
	})
}
