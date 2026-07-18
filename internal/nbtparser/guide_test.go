package nbtparser

import (
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

// guideTestRoot builds a minimal Create/vanilla structure NBT.
type guideTestRoot struct {
	Size        []int32           `nbt:"size"`
	Palette     []guideTestPal    `nbt:"palette"`
	Blocks      []guideTestBlock  `nbt:"blocks"`
	Entities    []guideTestEntity `nbt:"entities"`
	DataVersion int32             `nbt:"DataVersion"`
}

type guideTestPal struct {
	Name string `nbt:"Name"`
}

type guideTestBlock struct {
	Pos   []int32 `nbt:"pos"`
	State int32   `nbt:"state"`
}

type guideTestEntity struct {
	Pos      []float64      `nbt:"pos"`
	BlockPos []int32        `nbt:"blockPos"`
	Nbt      map[string]any `nbt:"nbt"`
}

func marshalGuideTestNBT(t *testing.T, root guideTestRoot) []byte {
	t.Helper()
	data, err := nbt.Marshal(root)
	if err != nil {
		t.Fatalf("marshal test NBT: %v", err)
	}
	return data
}

// Some exporters (e.g. Create with Steam 'n' Rails) store entities whose
// positions are world-absolute or negated-world coordinates. Entities must
// not appear in the guide at all, and must not inflate its bounding box.
func Test_ExtractGuideBlocks_IgnoresEntities(t *testing.T) {
	data := marshalGuideTestNBT(t, guideTestRoot{
		Size:    []int32{2, 1, 2},
		Palette: []guideTestPal{{Name: "minecraft:stone"}},
		Blocks: []guideTestBlock{
			{Pos: []int32{0, 0, 0}, State: 0},
			{Pos: []int32{1, 0, 1}, State: 0},
		},
		Entities: []guideTestEntity{
			{
				Pos:      []float64{-20481019.02, -84.07, -20485043.92},
				BlockPos: []int32{-20481020, -85, -20485044},
				Nbt:      map[string]any{"id": "minecraft:armor_stand"},
			},
			{
				Pos:      []float64{1.5, 0.0, 1.5},
				BlockPos: []int32{1, 0, 1},
				Nbt:      map[string]any{"id": "minecraft:item_frame"},
			},
		},
		DataVersion: 3955,
	})

	g, err := ExtractGuideBlocks(data)
	if err != nil {
		t.Fatalf("ExtractGuideBlocks: %v", err)
	}
	if len(g.Blocks) != 2 {
		t.Fatalf("expected 2 blocks (entities excluded), got %d", len(g.Blocks))
	}
	if g.SizeX != 2 || g.SizeY != 1 || g.SizeZ != 2 {
		t.Errorf("size = %dx%dx%d, want 2x1x2", g.SizeX, g.SizeY, g.SizeZ)
	}
	for _, b := range g.Blocks {
		if b.X < 0 || b.X > 1 || b.Y != 0 || b.Z < 0 || b.Z > 1 {
			t.Errorf("block out of expected bounds: %+v", b)
		}
	}
}

func Test_ExtractGuideBlocks_DropsOutlierBlocks(t *testing.T) {
	blocks := []guideTestBlock{
		{Pos: []int32{0, 0, 0}, State: 0},
		{Pos: []int32{1, 0, 0}, State: 0},
		{Pos: []int32{2, 0, 1}, State: 0},
		// Corrupt stray block at world-absolute coordinates.
		{Pos: []int32{20481075, 85, 20485068}, State: 0},
	}
	data := marshalGuideTestNBT(t, guideTestRoot{
		Size:        []int32{3, 1, 2},
		Palette:     []guideTestPal{{Name: "minecraft:stone"}},
		Blocks:      blocks,
		DataVersion: 3955,
	})

	g, err := ExtractGuideBlocks(data)
	if err != nil {
		t.Fatalf("ExtractGuideBlocks: %v", err)
	}
	if len(g.Blocks) != 3 {
		t.Fatalf("expected 3 blocks (outlier dropped), got %d", len(g.Blocks))
	}
	if g.SizeX > 3 || g.SizeY > 1 || g.SizeZ > 2 {
		t.Errorf("size = %dx%dx%d, want at most 3x1x2", g.SizeX, g.SizeY, g.SizeZ)
	}
}

func Test_DropGuideOutliers_KeepsNormalBuilds(t *testing.T) {
	var raw []guideRawBlock
	for x := 0; x < 200; x++ {
		raw = append(raw, guideRawBlock{x: x, y: x % 30, z: x % 50})
	}
	kept := dropGuideOutliers(raw)
	if len(kept) != 200 {
		t.Errorf("normal build lost blocks: %d of 200 kept", len(kept))
	}
}
