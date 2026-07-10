package schematic

import (
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

func Test_IsCopycat(t *testing.T) {
	for name, want := range map[string]bool{
		"create:copycat_panel":  true,
		"create:copycat_step":   true,
		"copycats:copycat_slab": true,
		"copycats:copycat_beam": true,
		"create:cogwheel":       false,
		"minecraft:stone":       false,
	} {
		if IsCopycat(name) != want {
			t.Errorf("IsCopycat(%q) = %v, want %v", name, !want, want)
		}
	}
}

// copycatBE builds a copycat block entity with the given wrapped material.
func copycatBE(pos [3]int, id, material string) BlockEntity {
	fields := map[string]nbt.RawMessage{
		"id": rawString(id),
	}
	if material != "" {
		fields["Material"] = compoundFromFields(map[string]nbt.RawMessage{
			"Name": rawString(material),
		})
	}
	return BlockEntity{Pos: pos, Raw: compoundFromFields(fields)}
}

func Test_Materials_IncludeCopycatContents(t *testing.T) {
	s := New(4, 1, 1)
	panel := s.PaletteIndex(BlockState{Name: "create:copycat_panel"})
	slab := s.PaletteIndex(BlockState{Name: "copycats:copycat_slab"})
	stone := s.PaletteIndex(BlockState{Name: "minecraft:stone"})
	s.Blocks[s.Index(0, 0, 0)] = panel
	s.Blocks[s.Index(1, 0, 0)] = slab
	s.Blocks[s.Index(2, 0, 0)] = panel
	s.Blocks[s.Index(3, 0, 0)] = stone

	s.BlockEntities = append(s.BlockEntities,
		copycatBE([3]int{0, 0, 0}, "create:copycat", "minecraft:oak_planks"),
		copycatBE([3]int{1, 0, 0}, "copycats:copycat", "minecraft:oak_planks"),
		// No material applied yet: air must not appear in the list.
		copycatBE([3]int{2, 0, 0}, "create:copycat", "minecraft:air"),
		// Out-of-bounds block entity must be ignored, not panic.
		copycatBE([3]int{9, 0, 0}, "create:copycat", "minecraft:dirt"),
	)

	got := map[string]int{}
	for _, m := range s.Materials() {
		got[m.BlockID] = m.Count
	}
	want := map[string]int{
		"create:copycat_panel":  2,
		"copycats:copycat_slab": 1,
		"minecraft:stone":       1,
		"minecraft:oak_planks":  2,
	}
	for id, c := range want {
		if got[id] != c {
			t.Errorf("materials[%s] = %d, want %d", id, got[id], c)
		}
	}
	if _, ok := got["minecraft:air"]; ok {
		t.Errorf("air counted as material")
	}
	if _, ok := got["minecraft:dirt"]; ok {
		t.Errorf("out-of-bounds block entity counted")
	}
}

func Test_Materials_NonCopycatBlockEntityIgnored(t *testing.T) {
	s := New(1, 1, 1)
	chest := s.PaletteIndex(BlockState{Name: "minecraft:chest"})
	s.Blocks[0] = chest
	// A chest with a Material-looking tag must not add materials: only
	// copycat palette blocks are consulted.
	s.BlockEntities = append(s.BlockEntities, copycatBE([3]int{0, 0, 0}, "minecraft:chest", "minecraft:diamond_block"))
	for _, m := range s.Materials() {
		if m.BlockID == "minecraft:diamond_block" {
			t.Errorf("non-copycat block entity contributed a material")
		}
	}
}
