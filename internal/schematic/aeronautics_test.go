package schematic

import (
	"os"
	"testing"
)

// airship.excraft is a real Create: Aeronautics Toolgun blueprint (format
// enxv_aeronautics_plot_print_v8) — a small airship saved as chunk sections.
func loadExcraft(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("aeronauticsdata/airship.excraft")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func TestExcraft_Detect(t *testing.T) {
	f, err := Detect(loadExcraft(t))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if f != FormatExcraft {
		t.Fatalf("Detect = %q, want %q", f, FormatExcraft)
	}
}

func TestExcraft_Read(t *testing.T) {
	s, err := ReadExcraft(loadExcraft(t))
	if err != nil {
		t.Fatalf("ReadExcraft: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if s.Meta.Name != "airship" {
		t.Errorf("Name = %q, want airship", s.Meta.Name)
	}
	if s.Meta.SourceFormat != string(FormatExcraft) {
		t.Errorf("SourceFormat = %q", s.Meta.SourceFormat)
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
	// The blueprint's own material summary reports 3097 blocks; extraction
	// should land in that ballpark (sails/doors count slightly differently).
	if nonAir < 3000 || nonAir > 3200 {
		t.Errorf("non-air blocks = %d, want ~3097", nonAir)
	}
	// Block entities must be present and remapped into bounds (this regressed
	// when section-Y offset was wrong — every BE fell outside the box).
	if len(s.BlockEntities) < 500 {
		t.Errorf("block entities = %d, want ~747", len(s.BlockEntities))
	}
	for _, be := range s.BlockEntities {
		for a := 0; a < 3; a++ {
			if be.Pos[a] < 0 || be.Pos[a] >= s.Size[a] {
				t.Fatalf("block entity %v outside bounds %v", be.Pos, s.Size)
			}
		}
	}
	// A known Create block entity should survive with its id intact.
	foundGearbox := false
	for _, be := range s.BlockEntities {
		var id struct {
			ID string `nbt:"id"`
		}
		if unmarshalRaw(be.Raw, &id) == nil && id.ID == "create:gearbox" {
			foundGearbox = true
			break
		}
	}
	if !foundGearbox {
		t.Error("expected a create:gearbox block entity")
	}
}

// The upload pipeline converts every non-structure format to Create .nbt, so
// the blueprint must convert and round-trip losslessly through structure NBT.
func TestExcraft_ConvertToStructure(t *testing.T) {
	src, err := ReadExcraft(loadExcraft(t))
	if err != nil {
		t.Fatalf("ReadExcraft: %v", err)
	}
	res, err := Convert(loadExcraft(t), FormatStructure)
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
