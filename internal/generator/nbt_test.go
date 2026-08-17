package generator

import "testing"

// The exporter must write the generated stair facing verbatim so the exported
// hull matches the frontend preview (generator.js). A previous 180° facing
// flip pointed every stair the wrong way, reading as inverted steps on hulls.
func TestBlockToPalette_StairFacingPreserved(t *testing.T) {
	mat := MaterialConfig{WoodType: "spruce"}
	for _, facing := range []string{"north", "south", "east", "west"} {
		for _, half := range []string{"top", "bottom"} {
			b := Block{Type: BlockStair, Props: map[string]string{
				"facing": facing, "half": half, "shape": "straight", "waterlogged": "false",
			}}
			_, props := blockToPalette(b, mat)
			if props["facing"] != facing {
				t.Errorf("facing %q/%q: exported facing = %q, want %q (unflipped)", facing, half, props["facing"], facing)
			}
			if props["half"] != half {
				t.Errorf("facing %q/%q: exported half = %q, want %q", facing, half, props["half"], half)
			}
		}
	}
}
