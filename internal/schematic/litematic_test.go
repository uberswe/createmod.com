package schematic

import (
	"reflect"
	"testing"
)

func Test_LitBitPack_RoundTrip(t *testing.T) {
	for _, paletteSize := range []int{2, 3, 4, 5, 17, 100, 4096} {
		bitsPer := litBitsFor(paletteSize)
		values := make([]int32, 1000)
		for i := range values {
			values[i] = int32((i * 7) % paletteSize)
		}
		longs := litPack(values, bitsPer)
		for i, want := range values {
			if got := litUnpack(longs, bitsPer, i); int32(got) != want {
				t.Fatalf("palette %d: entry %d = %d, want %d", paletteSize, i, got, want)
			}
		}
	}
	// Minimum of 2 bits even for tiny palettes.
	if litBitsFor(2) != 2 || litBitsFor(1) != 2 {
		t.Errorf("bits floor broken: %d %d", litBitsFor(2), litBitsFor(1))
	}
	if litBitsFor(5) != 3 || litBitsFor(4096) != 12 {
		t.Errorf("bits calc broken")
	}
}

func Test_Litematic_RoundTrip(t *testing.T) {
	for name, data := range map[string][]byte{
		"generator": generatorFixture(t),
		"handmade":  handmadeFixture(t),
	} {
		src, err := ReadStructureNBT(data)
		if err != nil {
			t.Fatalf("%s: read structure: %v", name, err)
		}
		src.Meta.Name = "Test Build"
		lit, err := WriteLitematic(src)
		if err != nil {
			t.Fatalf("%s: write litematic: %v", name, err)
		}
		back, err := ReadLitematic(lit)
		if err != nil {
			t.Fatalf("%s: read litematic: %v", name, err)
		}

		if src.Size != back.Size {
			t.Errorf("%s: size %v -> %v", name, src.Size, back.Size)
		}
		if src.DataVersion != back.DataVersion {
			t.Errorf("%s: DataVersion %d -> %d", name, src.DataVersion, back.DataVersion)
		}
		if back.Meta.Name != "Test Build" {
			t.Errorf("%s: name lost: %q", name, back.Meta.Name)
		}
		if !reflect.DeepEqual(src.Materials(), back.Materials()) {
			t.Errorf("%s: materials changed through .litematic round trip", name)
		}
		if len(src.BlockEntities) != len(back.BlockEntities) {
			t.Fatalf("%s: block entities %d -> %d", name, len(src.BlockEntities), len(back.BlockEntities))
		}
		for i := range src.BlockEntities {
			if src.BlockEntities[i].Pos != back.BlockEntities[i].Pos {
				t.Errorf("%s: block entity %d moved", name, i)
			}
			srcFields, _ := compoundFields(src.BlockEntities[i].Raw)
			backFields, _ := compoundFields(back.BlockEntities[i].Raw)
			if !reflect.DeepEqual(srcFields, backFields) {
				t.Errorf("%s: block entity %d payload changed", name, i)
			}
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

func Test_Litematic_NegativeSizeRegion(t *testing.T) {
	// Region with a negative Z size: Position marks the start corner and the
	// region extends toward -z; content must land normalized.
	src := New(1, 1, 2)
	src.DataVersion = 3120
	stone := src.PaletteIndex(BlockState{Name: "minecraft:stone"})
	src.Blocks[src.Index(0, 0, 0)] = stone

	root := litRootOut{
		MinecraftDataVersion: 3120,
		Version:              6,
		Metadata:             litMetadataOut{Name: "neg", EnclosingSize: litVec3{1, 1, 2}, RegionCount: 1},
		Regions: map[string]litRegionOut{
			"neg": {
				Position:          litVec3{0, 0, 1},
				Size:              litVec3{1, 1, -2},
				BlockStatePalette: []litPaletteEntryOut{{Name: "minecraft:air"}, {Name: "minecraft:stone"}},
				// order within region: index 0 = (x0,y0,z0-of-normalized)
				BlockStates: litPack([]int32{1, 0}, 2),
			},
		},
	}
	var buf = &[]byte{}
	_ = buf
	data, err := encodeAndGzip(root)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadLitematic(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if s.Size != [3]int{1, 1, 2} {
		t.Fatalf("size = %v", s.Size)
	}
	if s.At(0, 0, 0).Name != "minecraft:stone" {
		t.Errorf("normalized block placement wrong: %s at z0", s.At(0, 0, 0).Name)
	}
}
