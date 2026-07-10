package schematic

import (
	"testing"

	"createmod/internal/generator"
)

func hullModel(t *testing.T, wood string, length, beam, depth int) *Schematic {
	t.Helper()
	res, err := generator.GenerateHull(generator.HullParams{
		WoodType: wood, Length: length, Beam: beam, Depth: depth,
		BowLength: length / 4, SternLength: length / 6, BowSharpness: 1.3, SternSharpness: 0.7,
		SternStyle: "round", KeelCurve: 1.7, CastleBlend: 4, BottomPinch: 0.3,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := generator.ExportNBT(res)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadStructureNBT(data)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func Test_Fingerprint_GoldenPairs(t *testing.T) {
	base := hullModel(t, "spruce", 24, 8, 5)
	fpBase := ComputeFingerprint(base)

	// Identical build: ~100%
	same := Compare(fpBase, ComputeFingerprint(hullModel(t, "spruce", 24, 8, 5)))
	if same.Overall < 0.999 {
		t.Errorf("identical builds: %.3f, want ~1.0", same.Overall)
	}

	// Wood-swapped copy: same shape/function, different palette -> >= 0.85
	woodSwap := Compare(fpBase, ComputeFingerprint(hullModel(t, "dark_oak", 24, 8, 5)))
	if woodSwap.Overall < 0.85 {
		t.Errorf("wood-swapped: %.3f, want >= 0.85", woodSwap.Overall)
	}
	// ...and the breakdown must show WHERE they differ: palette low-ish,
	// shape near-identical.
	var shapeScore, paletteScore float64
	for _, c := range woodSwap.Components {
		switch c.Name {
		case "shape":
			shapeScore = c.Score
		case "palette":
			paletteScore = c.Score
		}
	}
	if shapeScore < 0.98 {
		t.Errorf("wood-swap shape = %.3f, want ~1", shapeScore)
	}
	if paletteScore > 0.5 {
		t.Errorf("wood-swap palette = %.3f, want low (different states)", paletteScore)
	}

	// Rotated copy (90° yaw): >= 0.95 via canonical yaw
	rot := hullModel(t, "spruce", 24, 8, 5)
	rotated := rotateModelY(t, rot)
	rotSim := Compare(fpBase, ComputeFingerprint(rotated))
	if rotSim.Overall < 0.95 {
		t.Errorf("rotated copy: %.3f, want >= 0.95", rotSim.Overall)
	}

	// Larger variant of the same design: clearly similar but not identical
	bigger := Compare(fpBase, ComputeFingerprint(hullModel(t, "spruce", 36, 12, 7)))
	if bigger.Overall < 0.55 || bigger.Overall > 0.98 {
		t.Errorf("scaled variant: %.3f, want in (0.55, 0.98)", bigger.Overall)
	}

	// Unrelated build (a solid stone box): < 0.6 overall, low shape score
	box := New(24, 8, 5)
	stone := box.PaletteIndex(BlockState{Name: "minecraft:stone"})
	for i := range box.Blocks {
		box.Blocks[i] = stone
	}
	unrelated := Compare(fpBase, ComputeFingerprint(box))
	if unrelated.Overall > 0.6 {
		t.Errorf("unrelated solid box: %.3f, want < 0.6", unrelated.Overall)
	}
	if unrelated.Overall >= woodSwap.Overall {
		t.Errorf("ranking broken: unrelated %.3f >= wood-swap %.3f", unrelated.Overall, woodSwap.Overall)
	}
}

// rotateModelY rotates a model 90° around Y (x,z -> z, sx-1-x).
func rotateModelY(t *testing.T, s *Schematic) *Schematic {
	t.Helper()
	out := New(s.Size[2], s.Size[1], s.Size[0])
	out.DataVersion = s.DataVersion
	out.Palette = append([]BlockState{}, s.Palette...)
	for y := 0; y < s.Size[1]; y++ {
		for z := 0; z < s.Size[2]; z++ {
			for x := 0; x < s.Size[0]; x++ {
				// note: block facing properties are NOT rotated — shape grid
				// doesn't care, palette hashes shift slightly, which is the
				// realistic case for a re-oriented reupload.
				out.Blocks[out.Index(z, y, s.Size[0]-1-x)] = s.Blocks[s.Index(x, y, z)]
			}
		}
	}
	return out
}

func Test_Fingerprint_Components_Weights(t *testing.T) {
	total := 0.0
	for _, cw := range componentWeights {
		total += cw.weight
	}
	if total < 0.999 || total > 1.001 {
		t.Fatalf("weights sum to %.3f", total)
	}
	fp := ComputeFingerprint(hullModel(t, "spruce", 24, 8, 5))
	sim := Compare(fp, fp)
	if sim.Overall < 0.999 {
		t.Errorf("self-similarity %.3f", sim.Overall)
	}
	if len(sim.Components) != 5 {
		t.Errorf("components = %d", len(sim.Components))
	}
}

func Test_Fingerprint_EncodeDecode(t *testing.T) {
	fp := ComputeFingerprint(hullModel(t, "spruce", 24, 8, 5))
	data, err := EncodeFingerprint(fp)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 8*1024 {
		t.Errorf("fingerprint too large: %d bytes", len(data))
	}
	back, err := DecodeFingerprint(data)
	if err != nil {
		t.Fatal(err)
	}
	sim := Compare(fp, back)
	if sim.Overall < 0.999 {
		t.Errorf("round-tripped fingerprint differs: %.3f", sim.Overall)
	}
}

func Test_BlockFamily(t *testing.T) {
	cases := map[string]string{
		"minecraft:oak_planks":        "planks",
		"minecraft:spruce_planks":     "planks",
		"minecraft:oak_stairs":        "stairs",
		"minecraft:stone":             "stone",
		"minecraft:deepslate_tiles":   "stone",
		"minecraft:chest":             "storage",
		"minecraft:redstone_lamp":     "redstone",
		"create:cogwheel":             "create_rotation",
		"create:shaft":                "create_rotation",
		"create:mechanical_press":     "create_processing",
		"create:water_wheel":          "create_power",
		"create:andesite_casing":      "create_casing_deco",
		"create:mechanical_bearing":   "create_movement",
		"somemod:weird_block":         "modded_other",
		"minecraft:glass_pane":        "glass",
	}
	for name, want := range cases {
		if got := BlockFamily(name); got != want {
			t.Errorf("%s -> %s, want %s", name, got, want)
		}
	}
}
