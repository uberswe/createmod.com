package schematic

import (
	"strings"
	"testing"
)

func editFixture(t *testing.T) *Schematic {
	t.Helper()
	s, err := ReadStructureNBT(handmadeFixture(t)) // 2x2x2: stone, stairs(E), chest+BE
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func Test_Ops_CropExpand(t *testing.T) {
	s := editFixture(t)
	cropped, err := ApplyOp(s, Op{Type: "crop", Min: [3]int{0, 0, 0}, Max: [3]int{0, 1, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if cropped.Size != [3]int{1, 2, 1} {
		t.Fatalf("crop size = %v", cropped.Size)
	}
	if cropped.At(0, 0, 0).Name != "minecraft:stone" || cropped.At(0, 1, 0).Name != "minecraft:chest" {
		t.Errorf("crop content wrong")
	}
	if len(cropped.BlockEntities) != 1 || cropped.BlockEntities[0].Pos != [3]int{0, 1, 0} {
		t.Errorf("crop BE: %+v", cropped.BlockEntities)
	}

	var op Op
	op.Type = "expand"
	op.Grow.Low = [3]int{1, 0, 2}
	op.Grow.High = [3]int{0, 3, 0}
	grown, err := ApplyOp(cropped, op)
	if err != nil {
		t.Fatal(err)
	}
	if grown.Size != [3]int{2, 5, 3} {
		t.Fatalf("expand size = %v", grown.Size)
	}
	if grown.At(1, 0, 2).Name != "minecraft:stone" {
		t.Errorf("expand shifted content wrong: %s", grown.At(1, 0, 2).Name)
	}
	if grown.BlockEntities[0].Pos != [3]int{1, 1, 2} {
		t.Errorf("expand BE pos: %v", grown.BlockEntities[0].Pos)
	}
}

func Test_Ops_RotateMirror(t *testing.T) {
	s := editFixture(t)
	// stairs at (1,0,0) facing east
	r, err := ApplyOp(s, Op{Type: "rotate", Steps: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Size != [3]int{2, 2, 2} {
		t.Fatalf("rotate size = %v", r.Size)
	}
	// (x=1,z=0) -> (nx=size_z-1-z=1, nz=x=1); facing east -> south
	if got := r.At(1, 0, 1); got.Name != "minecraft:oak_stairs" || got.Properties["facing"] != "south" {
		t.Errorf("rotated stairs: %+v", got)
	}
	// four rotations = identity
	r4 := s
	for i := 0; i < 4; i++ {
		r4, err = ApplyOp(r4, Op{Type: "rotate", Steps: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range s.Blocks {
		if s.Palette[s.Blocks[i]].Key() != r4.Palette[r4.Blocks[i]].Key() {
			t.Fatalf("rotate x4 not identity at %d", i)
		}
	}

	m, err := ApplyOp(s, Op{Type: "mirror", Axis: "x"})
	if err != nil {
		t.Fatal(err)
	}
	// stairs move from x=1 to x=0, facing east -> west
	if got := m.At(0, 0, 0); got.Name != "minecraft:oak_stairs" || got.Properties["facing"] != "west" {
		t.Errorf("mirrored stairs: %+v", got)
	}
	// mirror twice = identity
	m2, _ := ApplyOp(m, Op{Type: "mirror", Axis: "x"})
	for i := range s.Blocks {
		if s.Palette[s.Blocks[i]].Key() != m2.Palette[m2.Blocks[i]].Key() {
			t.Fatalf("mirror x2 not identity at %d", i)
		}
	}
}

func Test_Ops_FillHollowReplaceDelete(t *testing.T) {
	s := New(4, 4, 4)
	s.DataVersion = 3955
	filled, err := ApplyOp(s, Op{Type: "fill", Min: [3]int{0, 0, 0}, Max: [3]int{3, 3, 3}, Block: "minecraft:stone"})
	if err != nil {
		t.Fatal(err)
	}
	if filled.BlockCount() != 64 {
		t.Fatalf("fill count = %d", filled.BlockCount())
	}
	hollowed, err := ApplyOp(filled, Op{Type: "hollow", Min: [3]int{0, 0, 0}, Max: [3]int{3, 3, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if hollowed.BlockCount() != 64-8 {
		t.Errorf("hollow count = %d, want 56", hollowed.BlockCount())
	}
	if hollowed.At(1, 1, 1).Name != "minecraft:air" || hollowed.At(0, 0, 0).Name != "minecraft:stone" {
		t.Errorf("hollow wrong")
	}
	replaced, err := ApplyOp(hollowed, Op{Type: "replace", Replacements: []OpReplacement{{From: "minecraft:stone", To: "minecraft:oak_planks"}}})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.At(0, 0, 0).Name != "minecraft:oak_planks" {
		t.Errorf("replace wrong: %s", replaced.At(0, 0, 0).Name)
	}
	deleted, err := ApplyOp(replaced, Op{Type: "delete_region", Min: [3]int{0, 0, 0}, Max: [3]int{3, 0, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if deleted.At(0, 0, 0).Name != "minecraft:air" {
		t.Errorf("delete wrong")
	}

	// replace preserving properties within a family
	st := editFixture(t)
	swapped, err := ApplyOp(st, Op{Type: "replace", Replacements: []OpReplacement{{From: "minecraft:oak_stairs", To: "minecraft:spruce_stairs"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := swapped.At(1, 0, 0); got.Name != "minecraft:spruce_stairs" || got.Properties["facing"] != "east" {
		t.Errorf("family replace lost properties: %+v", got)
	}
	// removal drops the block AND its block entity
	removed, err := ApplyOp(st, Op{Type: "replace", Replacements: []OpReplacement{{From: "minecraft:chest", To: ""}}})
	if err != nil {
		t.Fatal(err)
	}
	if removed.At(0, 1, 0).Name != "minecraft:air" || len(removed.BlockEntities) != 0 {
		t.Errorf("removal: %s BEs=%d", removed.At(0, 1, 0).Name, len(removed.BlockEntities))
	}
}

func Test_Ops_ReplaySafety(t *testing.T) {
	s := editFixture(t)
	ops := []Op{
		{Type: "rotate", Steps: 1},
		{Type: "fill", Min: [3]int{0, 0, 0}, Max: [3]int{0, 0, 0}, Block: "minecraft:diamond_block"},
		{Type: "mirror", Axis: "z"},
	}
	a, err := ApplyOps(s, ops)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ApplyOps(s, ops)
	if err != nil {
		t.Fatal(err)
	}
	// deterministic replay
	outA, _ := WriteStructureNBT(a)
	outB, _ := WriteStructureNBT(b)
	if string(outA) != string(outB) {
		t.Errorf("replay not deterministic")
	}
	// source untouched by ops (purity)
	if s.At(1, 0, 0).Name != "minecraft:oak_stairs" || len(s.BlockEntities) != 1 {
		t.Errorf("source mutated by replay")
	}
	// unknown op and injection-ish block ids rejected
	if _, err := ApplyOp(s, Op{Type: "nope"}); err == nil {
		t.Errorf("unknown op accepted")
	}
	if _, err := ApplyOp(s, Op{Type: "fill", Max: [3]int{1, 1, 1}, Block: "minecraft:stone{Cmd}"}); err == nil {
		t.Errorf("bad block id accepted")
	}
	if !strings.Contains(func() string {
		_, err := ApplyOps(s, make([]Op, MaxOpsPerSession+1))
		return err.Error()
	}(), "op log") {
		t.Errorf("op log cap missing")
	}
}
