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
		"minecraft:oak_planks":      "planks",
		"minecraft:spruce_planks":   "planks",
		"minecraft:oak_stairs":      "stairs",
		"minecraft:stone":           "stone",
		"minecraft:deepslate_tiles": "stone",
		"minecraft:chest":           "storage",
		"minecraft:redstone_lamp":   "redstone",
		"create:cogwheel":           "create_rotation",
		"create:shaft":              "create_rotation",
		"create:mechanical_press":   "create_processing",
		"create:water_wheel":        "create_power",
		"create:andesite_casing":    "create_casing_deco",
		"create:mechanical_bearing": "create_movement",
		"somemod:weird_block":       "modded_other",
		"minecraft:glass_pane":      "glass",
	}
	for name, want := range cases {
		if got := BlockFamily(name); got != want {
			t.Errorf("%s -> %s, want %s", name, got, want)
		}
	}
}

// compositionModel builds a box filled with the given block counts (laid out
// linearly; remaining volume stays air). Composition is what matters for the
// materials/function components under test.
func compositionModel(t *testing.T, counts map[string]int) *Schematic {
	t.Helper()
	total := 0
	for _, n := range counts {
		total += n
	}
	edge := 1
	for edge*edge*edge < total {
		edge++
	}
	s := New(edge, edge, edge)
	i := 0
	for name, n := range counts {
		idx := s.PaletteIndex(BlockState{Name: name})
		for k := 0; k < n; k++ {
			s.Blocks[i] = idx
			i++
		}
	}
	return s
}

// A house with a few decorative Create blocks must not "function" like a
// machine: cosine direction alone is scale-invariant, which used to rank
// steam boilers as the most similar builds to houses.
func Test_Fingerprint_HouseDoesNotMatchMachine(t *testing.T) {
	house := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":       2000,
		"minecraft:oak_planks": 500,
		"minecraft:oak_stairs": 300,
		"minecraft:glass":      200,
		"create:shaft":         2,
		"create:fluid_pipe":    3,
	}))
	otherHouse := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":          1800,
		"minecraft:spruce_planks": 600,
		"minecraft:spruce_stairs": 250,
		"minecraft:glass":         150,
	}))
	boiler := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":      2000,
		"create:shaft":        60,
		"create:fluid_pipe":   120,
		"create:fluid_tank":   80,
		"create:steam_engine": 20,
	}))

	component := func(sim Similarity, name string) float64 {
		for _, c := range sim.Components {
			if c.Name == name {
				return c.Score
			}
		}
		return -1
	}

	vsBoiler := Compare(house, boiler)
	vsHouse := Compare(house, otherHouse)

	if f := component(vsBoiler, "function"); f > 0.3 {
		t.Errorf("house vs boiler function = %.2f, want <= 0.3 (house is ~0.2%% machinery, boiler ~11%%)", f)
	}
	// Two non-machines agree on function even when one has a couple of
	// decorative Create blocks.
	if f := component(vsHouse, "function"); f < 0.99 {
		t.Errorf("house vs house function = %.2f, want ~1", f)
	}
	// Materials must not be hijacked by the shared terrain (dirt) majority.
	if m := component(vsBoiler, "materials"); m > 0.85 {
		t.Errorf("house vs boiler materials = %.2f, want < 0.85 despite shared dirt majority", m)
	}
	if vsHouse.Overall <= vsBoiler.Overall {
		t.Errorf("ranking broken: house-vs-house %.2f <= house-vs-boiler %.2f", vsHouse.Overall, vsBoiler.Overall)
	}
}

func Test_Fingerprint_MachinesStillMatchMachines(t *testing.T) {
	boilerA := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":      500,
		"create:shaft":        60,
		"create:fluid_pipe":   120,
		"create:fluid_tank":   80,
		"create:steam_engine": 20,
	}))
	boilerB := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":      400,
		"create:shaft":        50,
		"create:fluid_pipe":   140,
		"create:fluid_tank":   70,
		"create:steam_engine": 25,
	}))
	for _, c := range Compare(boilerA, boilerB).Components {
		if c.Name == "function" && c.Score < 0.8 {
			t.Errorf("boiler vs boiler function = %.2f, want >= 0.8", c.Score)
		}
	}
}

// The function component's weight scales with how much machinery is present:
// a pure-decoration pair gets zero function weight (the trivial "both
// non-machines" agreement can no longer drown the shape signal), a lone
// functional block gets a couple of percent, and machinery-heavy pairs get
// the full nominal weight. Weights always renormalize to 1.
func Test_Fingerprint_FunctionWeightScales(t *testing.T) {
	pureHouse := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":       2000,
		"minecraft:oak_planks": 500,
	}))
	otherPureHouse := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":          1800,
		"minecraft:spruce_planks": 600,
	}))
	bearingHouse := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":            2000,
		"minecraft:oak_planks":      500,
		"create:mechanical_bearing": 4,
	}))
	boiler := ComputeFingerprint(compositionModel(t, map[string]int{
		"minecraft:dirt":      2000,
		"create:shaft":        60,
		"create:fluid_pipe":   120,
		"create:fluid_tank":   80,
		"create:steam_engine": 20,
	}))

	weight := func(sim Similarity, name string) float64 {
		for _, c := range sim.Components {
			if c.Name == name {
				return c.Weight
			}
		}
		return -1
	}
	sumWeights := func(sim Similarity) float64 {
		total := 0.0
		for _, c := range sim.Components {
			total += c.Weight
		}
		return total
	}

	houses := Compare(pureHouse, otherPureHouse)
	if w := weight(houses, "function"); w != 0 {
		t.Errorf("pure house pair function weight = %.4f, want 0", w)
	}
	withBearing := Compare(pureHouse, bearingHouse)
	if w := weight(withBearing, "function"); w <= 0 || w > 0.06 {
		t.Errorf("house vs bearing-house function weight = %.4f, want small but non-zero", w)
	}
	machines := Compare(boiler, boiler)
	if w := weight(machines, "function"); w < 0.15 {
		t.Errorf("machine pair function weight = %.4f, want near the nominal 0.20", w)
	}
	for name, sim := range map[string]Similarity{"houses": houses, "bearing": withBearing, "machines": machines} {
		if s := sumWeights(sim); s < 0.999 || s > 1.001 {
			t.Errorf("%s: weights sum to %.4f, want 1", name, s)
		}
	}
}
