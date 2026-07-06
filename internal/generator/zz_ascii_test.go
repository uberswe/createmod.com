package generator

import (
	"fmt"
	"os"
	"testing"
)

func TestASCII(t *testing.T) {
	if os.Getenv("ASCII") == "" {
		t.Skip("set ASCII")
	}
	p := HullParams{
		Version: 3, WoodType: "dark_oak", Length: 72, Beam: 22, Depth: 13,
		BottomPinch: 0.35, HullFlare: 0.4, FlareCurve: 2.4, Tumblehome: 0.35, TumbleCurve: 2.8,
		SheerCurve: 0.22, SheerCurveExp: 2, BowLength: 16, BowSharpness: 1.5,
		BowKeelRise: 0.7, BowKeelLength: 16, BowCurve: 0.15, SternStyle: "square",
		SternLength: 9, SternSharpness: 0.5, SternKeelRise: 0.4, SternKeelLength: 10,
		SternOverhang: 0.55, KeelCurve: 1.7, MidWidthBias: 0.2, CastleBlend: 6,
		CastleHeight: 5, CastleLength: 18, ForecastleHeight: 2, ForecastleLength: 9,
		BowStyle: "default", StemRake: 0.55, StemCurve: 0.3, SternRake: 0.2, Rocker: 0.05,
		Deadrise: 0.1, MidFullness: 0.7, BowSectionV: 0.5, SternFullness: 0.6, ParallelMidbody: 0.25,
	}
	r, err := GenerateHull(p)
	if err != nil {
		t.Fatal(err)
	}
	// map with block-type glyphs: # plank, s stair, _ slab-bottom, - slab-top
	glyph := map[int]string{BlockPlank: "#", BlockStair: "s", BlockSlabBot: "_", BlockSlabTop: "-", BlockFence: "f", BlockTrapdoor: "t"}
	occ := map[[3]int]string{}
	maxX, maxY, maxZ := 0, 0, 0
	for _, b := range r.Blocks {
		g, ok := glyph[b.Type]
		if !ok { g = "?" }
		occ[[3]int{b.X, b.Y, b.Z}] = g
		if b.X > maxX { maxX = b.X }
		if b.Y > maxY { maxY = b.Y }
		if b.Z > maxZ { maxZ = b.Z }
	}
	fmt.Printf("blocks=%d size=%dx%dx%d\n", len(r.Blocks), r.SizeX, r.SizeY, r.SizeZ)
	fmt.Println("== midship cross-section ==")
	zm := maxZ / 2
	for y := maxY; y >= 0; y-- {
		row := ""
		for x := 0; x <= maxX; x++ {
			if g, ok := occ[[3]int{x, y, zm}]; ok { row += g } else { row += "." }
		}
		fmt.Println(row)
	}
	fmt.Println("== bow quarter section (z = 80%) ==")
	zb := maxZ * 4 / 5
	for y := maxY; y >= 0; y-- {
		row := ""
		for x := 0; x <= maxX; x++ {
			if g, ok := occ[[3]int{x, y, zb}]; ok { row += g } else { row += "." }
		}
		fmt.Println(row)
	}
}
