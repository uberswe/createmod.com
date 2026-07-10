package schematic

import (
	"strings"
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

// legacyFixture builds a 3x1x1 .schematic: stone, oak planks, red wool.
func legacyFixture(t *testing.T) []byte {
	t.Helper()
	root := struct {
		Width     int16  `nbt:"Width"`
		Height    int16  `nbt:"Height"`
		Length    int16  `nbt:"Length"`
		Materials string `nbt:"Materials"`
		Blocks    []byte `nbt:"Blocks"`
		Data      []byte `nbt:"Data"`
	}{
		Width: 3, Height: 1, Length: 1, Materials: "Alpha",
		Blocks: []byte{1, 5, 35},  // stone, planks, wool
		Data:   []byte{0, 0, 14}, // meta: plain, oak, red
	}
	data, err := encodeAndGzip(root)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func Test_Legacy_Read(t *testing.T) {
	s, err := ReadLegacy(legacyFixture(t))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if s.Size != [3]int{3, 1, 1} {
		t.Fatalf("size = %v", s.Size)
	}
	if got := s.At(0, 0, 0).Name; got != "minecraft:stone" {
		t.Errorf("1:0 -> %s", got)
	}
	if got := s.At(1, 0, 0).Name; got != "minecraft:oak_planks" {
		t.Errorf("5:0 -> %s", got)
	}
	if got := s.At(2, 0, 0).Name; got != "minecraft:red_wool" {
		t.Errorf("35:14 -> %s", got)
	}
	if s.DataVersion != legacyDataVersion {
		t.Errorf("DataVersion = %d", s.DataVersion)
	}
	if f, err := Detect(legacyFixture(t)); err != nil || f != FormatLegacy {
		t.Errorf("detect = %v %v", f, err)
	}
}

func Test_Legacy_WriteRoundTrip(t *testing.T) {
	// Vanilla pre-1.13 content must survive model -> .schematic -> model.
	src := New(2, 2, 1)
	src.DataVersion = 3955
	stone := src.PaletteIndex(BlockState{Name: "minecraft:stone"})
	wool := src.PaletteIndex(BlockState{Name: "minecraft:blue_wool"})
	modern := src.PaletteIndex(BlockState{Name: "minecraft:crimson_planks"}) // no legacy equivalent
	src.Blocks[src.Index(0, 0, 0)] = stone
	src.Blocks[src.Index(1, 0, 0)] = wool
	src.Blocks[src.Index(0, 1, 0)] = modern

	out, warnings, err := WriteLegacy(src)
	if err != nil {
		t.Fatal(err)
	}
	foundDrop := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "crimson_planks") {
			foundDrop = true
		}
	}
	if !foundDrop {
		t.Errorf("no warning about dropped crimson_planks; warnings: %v", warnings)
	}

	back, err := ReadLegacy(out)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got := back.At(0, 0, 0).Name; got != "minecraft:stone" {
		t.Errorf("stone -> %s", got)
	}
	if got := back.At(1, 0, 0).Name; got != "minecraft:blue_wool" {
		t.Errorf("blue_wool -> %s", got)
	}
	if got := back.At(0, 1, 0).Name; got != "minecraft:air" {
		t.Errorf("crimson_planks should drop to air, got %s", got)
	}
}

func Test_Legacy_ViaConvert(t *testing.T) {
	res, err := Convert(legacyFixture(t), FormatSponge)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if res.From != FormatLegacy {
		t.Errorf("detected %s", res.From)
	}
	if len(res.Warnings) == 0 {
		t.Errorf("legacy conversion produced no warnings")
	}
	s, err := ReadSponge(res.Data)
	if err != nil {
		t.Fatal(err)
	}
	if s.BlockCount() != 3 {
		t.Errorf("blocks = %d", s.BlockCount())
	}
}

// sableFixture builds a blueprint with two sub-levels and a block entity.
func sableFixture(t *testing.T) []byte {
	t.Helper()
	type vec3i struct {
		X int32 `nbt:"x"`
		Y int32 `nbt:"y"`
		Z int32 `nbt:"z"`
	}
	type palEntry struct {
		Name       string            `nbt:"Name"`
		Properties map[string]string `nbt:"Properties,omitempty"`
	}
	type blockBE struct {
		LocalPos vec3i `nbt:"local_pos"`
		Palette  int32 `nbt:"palette_id"`
		BEID     int32 `nbt:"block_entity_data_id"`
	}
	type block struct {
		LocalPos vec3i `nbt:"local_pos"`
		Palette  int32 `nbt:"palette_id"`
	}
	beRaw := compoundFromFields(map[string]nbt.RawMessage{
		"id": rawString("minecraft:chest"),
	})
	type subLevelA struct {
		ID           int32      `nbt:"id"`
		BlocksOrigin vec3i      `nbt:"blocks_origin"`
		Palette      []palEntry `nbt:"block_palette"`
		Blocks       []blockBE  `nbt:"blocks"`
		BEs          rawList    `nbt:"block_entities"`
	}
	type subLevelB struct {
		ID           int32      `nbt:"id"`
		BlocksOrigin vec3i      `nbt:"blocks_origin"`
		Palette      []palEntry `nbt:"block_palette"`
		Blocks       []block    `nbt:"blocks"`
		Unavailable  []int32    `nbt:"unavailable_palette_ids"`
		Entities     rawList    `nbt:"entities"`
	}
	ent := compoundFromFields(map[string]nbt.RawMessage{"id": rawString("create:contraption")})
	root := struct {
		Version   int32         `nbt:"version"`
		SubLevels []interface{} `nbt:"sub_levels"`
	}{
		Version: 1,
		SubLevels: []interface{}{
			subLevelA{
				ID:           0,
				BlocksOrigin: vec3i{10, 0, 10},
				Palette:      []palEntry{{Name: "minecraft:chest", Properties: map[string]string{"facing": "north"}}},
				Blocks:       []blockBE{{LocalPos: vec3i{0, 0, 0}, Palette: 0, BEID: 0}},
				BEs:          rawList{beRaw},
			},
			subLevelB{
				ID:           1,
				BlocksOrigin: vec3i{11, 0, 10},
				Palette:      []palEntry{{Name: "minecraft:stone"}, {Name: "modded:unknown_block"}},
				Blocks:       []block{{LocalPos: vec3i{0, 0, 0}, Palette: 0}, {LocalPos: vec3i{1, 0, 0}, Palette: 1}},
				Unavailable:  []int32{1},
				Entities:     rawList{ent},
			},
		},
	}
	data, err := encodeAndGzip(root)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func Test_Sable_Read(t *testing.T) {
	data := sableFixture(t)
	if f, err := Detect(data); err != nil || f != FormatSable {
		t.Fatalf("detect = %v %v", f, err)
	}
	s, err := ReadSable(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Blocks at world (10..12, 0, 10) -> normalized 3x1x1
	if s.Size != [3]int{3, 1, 1} {
		t.Fatalf("size = %v", s.Size)
	}
	if got := s.At(0, 0, 0).Name; got != "minecraft:chest" {
		t.Errorf("chest -> %s", got)
	}
	if got := s.At(1, 0, 0).Name; got != "minecraft:stone" {
		t.Errorf("stone -> %s", got)
	}
	if got := s.At(2, 0, 0).Name; got != "minecraft:air" {
		t.Errorf("unavailable palette entry should be air, got %s", got)
	}
	if len(s.BlockEntities) != 1 || s.BlockEntities[0].Pos != [3]int{0, 0, 0} {
		t.Fatalf("block entities = %+v", s.BlockEntities)
	}
	notes := strings.Join(s.Meta.LossyNotes, " | ")
	for _, want := range []string{"blocks only", "sub-levels merged", "entities"} {
		if !strings.Contains(notes, want) {
			t.Errorf("lossy notes missing %q: %s", want, notes)
		}
	}
	// Sable converts out but never in.
	if _, err := Convert(data, FormatStructure); err != nil {
		t.Errorf("sable -> structure: %v", err)
	}
	if _, err := Write(s, FormatSable); err == nil {
		t.Errorf("writing sable should be unsupported")
	}
}
