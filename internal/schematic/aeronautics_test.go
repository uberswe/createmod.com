package schematic

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

// buildTestExcraft synthesizes a minimal Create: Aeronautics Toolgun blueprint
// in memory so the parser stays covered without committing a binary fixture.
// It mirrors the on-disk format: gzip NBT with one chunk holding one 16^3
// section (MC 1.18 non-spanning bit-packed block_states) — a 4x4x4 stone cube
// with a create:gearbox at local (1,2,3) plus its block entity in world coords.
func buildTestExcraft(t *testing.T) []byte {
	t.Helper()
	const (
		stone   = 1
		gearbox = 2
		bits    = 4
		perLong = 64 / bits
	)
	// Section indices are laid out as i = lx + lz*16 + ly*256, matching the
	// lx=i&15, lz=(i>>4)&15, ly=(i>>8)&15 decode in ReadExcraft.
	const mask = (1 << bits) - 1
	data := make([]int64, 4096/perLong)
	set := func(lx, ly, lz, val int) {
		i := lx + lz*16 + ly*256
		li := i / perLong
		off := uint((i % perLong) * bits)
		data[li] = (data[li] &^ (int64(mask) << off)) | (int64(val) << off)
	}
	for lx := 0; lx < 4; lx++ {
		for ly := 0; ly < 4; ly++ {
			for lz := 0; lz < 4; lz++ {
				set(lx, ly, lz, stone)
			}
		}
	}
	set(1, 2, 3, gearbox)

	// Encode-side mirrors of the read structs; block_entities carry inline
	// world x/y/z (stripped on read).
	type encBE struct {
		ID string `nbt:"id"`
		X  int32  `nbt:"x"`
		Y  int32  `nbt:"y"`
		Z  int32  `nbt:"z"`
	}
	type encChunk struct {
		Sections      map[string]excraftSection `nbt:"sections"`
		BlockEntities []encBE                   `nbt:"block_entities"`
	}
	type encPlot struct {
		Chunks map[string]encChunk `nbt:"chunks"`
	}
	type encSubLevel struct {
		Plot encPlot `nbt:"plot"`
	}
	type encRoot struct {
		Format               string        `nbt:"format"`
		Name                 string        `nbt:"name"`
		SourceMinBuildHeight int32         `nbt:"source_min_build_height"`
		SubLevels            []encSubLevel `nbt:"sublevels"`
	}

	section := excraftSection{BlockStates: excraftBlockStates{
		Palette: []excraftPaletteEntry{
			{Name: "minecraft:air"},
			{Name: "minecraft:stone"},
			{Name: "create:gearbox"},
		},
		Data: data,
	}}
	// minBuildHeight -64 -> secOffset -4, so section key "4" is world section 0
	// (world Y 0..15) and local (lx,ly,lz) lands at world (lx,ly,lz).
	root := encRoot{
		Format:               formatExcraftV8,
		Name:                 "test-ship",
		SourceMinBuildHeight: -64,
		SubLevels: []encSubLevel{{Plot: encPlot{Chunks: map[string]encChunk{
			"0": {
				Sections:      map[string]excraftSection{"4": section},
				BlockEntities: []encBE{{ID: "create:gearbox", X: 1, Y: 2, Z: 3}},
			},
		}}}},
	}

	var raw bytes.Buffer
	if err := nbt.NewEncoder(&raw).Encode(root, ""); err != nil {
		t.Fatalf("encode nbt: %v", err)
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(raw.Bytes()); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return gz.Bytes()
}

func TestExcraft_Detect(t *testing.T) {
	f, err := Detect(buildTestExcraft(t))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if f != FormatExcraft {
		t.Fatalf("Detect = %q, want %q", f, FormatExcraft)
	}
}

func TestExcraft_Read(t *testing.T) {
	s, err := ReadExcraft(buildTestExcraft(t))
	if err != nil {
		t.Fatalf("ReadExcraft: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if s.Meta.Name != "test-ship" {
		t.Errorf("Name = %q, want test-ship", s.Meta.Name)
	}
	if s.Meta.SourceFormat != string(FormatExcraft) {
		t.Errorf("SourceFormat = %q", s.Meta.SourceFormat)
	}
	// The 4x4x4 cube lands at world (0..3) on every axis.
	if s.Size != [3]int{4, 4, 4} {
		t.Errorf("Size = %v, want [4 4 4]", s.Size)
	}
	// palette[0] must be air so unset cells read as air.
	if !s.Palette[0].IsAir() {
		t.Errorf("palette[0] = %q, want air", s.Palette[0].Name)
	}

	nonAir := 0
	for _, idx := range s.Blocks {
		if !s.Palette[idx].IsAir() {
			nonAir++
		}
	}
	if nonAir != 64 {
		t.Errorf("non-air blocks = %d, want 64", nonAir)
	}
	// Block entities must be present and remapped into bounds (this regressed
	// when the section-Y offset was wrong — every BE fell outside the box).
	if len(s.BlockEntities) != 1 {
		t.Fatalf("block entities = %d, want 1", len(s.BlockEntities))
	}
	for _, be := range s.BlockEntities {
		for a := 0; a < 3; a++ {
			if be.Pos[a] < 0 || be.Pos[a] >= s.Size[a] {
				t.Fatalf("block entity %v outside bounds %v", be.Pos, s.Size)
			}
		}
	}
	// The Create block entity must survive with its id intact.
	var id struct {
		ID string `nbt:"id"`
	}
	if err := unmarshalRaw(s.BlockEntities[0].Raw, &id); err != nil || id.ID != "create:gearbox" {
		t.Errorf("block entity id = %q (err %v), want create:gearbox", id.ID, err)
	}
}

// The upload pipeline converts every non-structure format to Create .nbt, so
// the blueprint must convert and round-trip losslessly through structure NBT.
func TestExcraft_ConvertToStructure(t *testing.T) {
	data := buildTestExcraft(t)
	src, err := ReadExcraft(data)
	if err != nil {
		t.Fatalf("ReadExcraft: %v", err)
	}
	res, err := Convert(data, FormatStructure)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if res.From != FormatExcraft {
		t.Fatalf("From = %q, want excraft", res.From)
	}
	got, err := ReadStructureNBT(res.Data)
	if err != nil {
		t.Fatalf("re-read structure: %v", err)
	}
	if got.Size != src.Size {
		t.Errorf("round-trip size %v != %v", got.Size, src.Size)
	}
	countNonAir := func(s *Schematic) int {
		n := 0
		for _, idx := range s.Blocks {
			if !s.Palette[idx].IsAir() {
				n++
			}
		}
		return n
	}
	if a, b := countNonAir(src), countNonAir(got); a != b {
		t.Errorf("round-trip non-air blocks %d != %d", b, a)
	}
	if len(got.BlockEntities) != len(src.BlockEntities) {
		t.Errorf("round-trip block entities %d != %d", len(got.BlockEntities), len(src.BlockEntities))
	}
}
