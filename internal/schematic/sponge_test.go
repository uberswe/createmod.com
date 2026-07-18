package schematic

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_Sponge_RoundTrip(t *testing.T) {
	for name, data := range map[string][]byte{
		"generator": generatorFixture(t),
		"handmade":  handmadeFixture(t),
	} {
		src, err := ReadStructureNBT(data)
		if err != nil {
			t.Fatalf("%s: read structure: %v", name, err)
		}
		schem, err := WriteSponge(src)
		if err != nil {
			t.Fatalf("%s: write sponge: %v", name, err)
		}
		back, err := ReadSponge(schem)
		if err != nil {
			t.Fatalf("%s: read sponge: %v", name, err)
		}

		if src.Size != back.Size {
			t.Errorf("%s: size %v -> %v", name, src.Size, back.Size)
		}
		if src.DataVersion != back.DataVersion {
			t.Errorf("%s: DataVersion %d -> %d", name, src.DataVersion, back.DataVersion)
		}
		if !reflect.DeepEqual(src.Materials(), back.Materials()) {
			t.Errorf("%s: materials changed through .schem round trip", name)
		}
		if len(src.BlockEntities) != len(back.BlockEntities) {
			t.Fatalf("%s: block entities %d -> %d", name, len(src.BlockEntities), len(back.BlockEntities))
		}
		for i := range src.BlockEntities {
			if src.BlockEntities[i].Pos != back.BlockEntities[i].Pos {
				t.Errorf("%s: block entity %d moved %v -> %v", name, i, src.BlockEntities[i].Pos, back.BlockEntities[i].Pos)
			}
			// Canonical payloads must survive the v3 Data nesting round trip.
			srcFields, _ := compoundFields(src.BlockEntities[i].Raw)
			backFields, _ := compoundFields(back.BlockEntities[i].Raw)
			if !reflect.DeepEqual(srcFields, backFields) {
				t.Errorf("%s: block entity %d payload changed: %v -> %v", name, i, srcFields, backFields)
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

func Test_Sponge_WriteByteStable(t *testing.T) {
	s, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := WriteSponge(s)
	if err != nil {
		t.Fatal(err)
	}
	b, err := WriteSponge(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf(".schem writer output is not deterministic")
	}
}

func Test_Sponge_ReadV2(t *testing.T) {
	// Hand-build a minimal v2 file: 2x1x1, stone then air.
	var data bytes.Buffer
	putUvarint(&data, 0) // stone
	putUvarint(&data, 1) // air
	root := struct {
		Version     int32            `nbt:"Version"`
		DataVersion int32            `nbt:"DataVersion"`
		Width       int16            `nbt:"Width"`
		Height      int16            `nbt:"Height"`
		Length      int16            `nbt:"Length"`
		Palette     orderedPalette   `nbt:"Palette"`
		BlockData   []byte           `nbt:"BlockData"`
	}{
		Version: 2, DataVersion: 3120, Width: 2, Height: 1, Length: 1,
		Palette:   orderedPalette{"minecraft:stone", "minecraft:air"},
		BlockData: data.Bytes(),
	}
	var buf bytes.Buffer
	if err := encodeNBT(&buf, root); err != nil {
		t.Fatal(err)
	}
	s, err := ReadSponge(buf.Bytes())
	if err != nil {
		t.Fatalf("read v2: %v", err)
	}
	if s.Size != [3]int{2, 1, 1} {
		t.Errorf("size = %v", s.Size)
	}
	if s.DataVersion != 3120 {
		t.Errorf("DataVersion = %d", s.DataVersion)
	}
	if got := s.At(0, 0, 0).Name; got != "minecraft:stone" {
		t.Errorf("block 0 = %s", got)
	}
	if got := s.At(1, 0, 0).Name; got != "minecraft:air" {
		t.Errorf("block 1 = %s", got)
	}
	if s.Meta.SourceFormat != "schem-v2" {
		t.Errorf("source = %s", s.Meta.SourceFormat)
	}
}

func Test_ParseStateString(t *testing.T) {
	st, err := ParseStateString("minecraft:oak_stairs[facing=east,half=bottom]")
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "minecraft:oak_stairs" || st.Properties["facing"] != "east" || st.Properties["half"] != "bottom" {
		t.Errorf("parsed %+v", st)
	}
	if st.Key() != "minecraft:oak_stairs[facing=east,half=bottom]" {
		t.Errorf("key round trip = %s", st.Key())
	}
	if _, err := ParseStateString("minecraft:stone[broken"); err == nil {
		t.Errorf("unterminated accepted")
	}
	plain, err := ParseStateString("minecraft:stone")
	if err != nil || plain.Name != "minecraft:stone" || plain.Properties != nil {
		t.Errorf("plain parse: %+v %v", plain, err)
	}
}
