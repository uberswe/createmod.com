package render_test

import (
	"fmt"
	"image/png"
	"os"
	"testing"

	"createmod/internal/generator"
	"createmod/internal/generator/render"
)

func saveHull(t *testing.T, name string, p generator.HullParams) {
	r, err := generator.GenerateHull(p)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	img := render.Isometric(r)
	f, _ := os.Create(name)
	png.Encode(f, img)
	f.Close()
	fmt.Printf("%s: %d blocks %dx%dx%d\n", name, len(r.Blocks), r.SizeX, r.SizeY, r.SizeZ)
}

func TestVisualCompare(t *testing.T) {
	out := os.Getenv("HULL_OUT")
	if out == "" {
		t.Skip("set HULL_OUT")
	}
	base := generator.HullParams{
		WoodType: "spruce", Length: 60, Beam: 13, Depth: 9,
		BottomPinch: 0.3, HullFlare: 0.2, FlareCurve: 2.0, Tumblehome: 0.05, TumbleCurve: 2.5,
		SheerCurve: 0.3, SheerCurveExp: 2.2, BowLength: 18, BowSharpness: 1.5,
		BowKeelRise: 0.6, BowKeelLength: 14, SternStyle: "round", SternLength: 10,
		SternSharpness: 0.8, SternKeelRise: 0.3, SternKeelLength: 7, KeelCurve: 1.7,
		CastleBlend: 4, HasRailings: true, HasTrim: true, BowStyle: "clipper", BowCurve: -0.4,
	}
	v1 := base
	v1.Version = 2
	saveHull(t, out+"/v1_clipper.png", v1)
	v2 := base
	v2.Version = 3
	saveHull(t, out+"/v2_clipper.png", v2)
	v2r := base
	v2r.Version = 3
	v2r.StemRake = 0.9
	v2r.StemCurve = -0.7
	v2r.SternRake = 0.5
	v2r.Rocker = 0.15
	v2r.Deadrise = 0.3
	v2r.MidFullness = 0.55
	v2r.BowSectionV = 0.7
	v2r.ParallelMidbody = 0.25
	saveHull(t, out+"/v2_tuned.png", v2r)
	longship := generator.HullParams{
		Version: 3, WoodType: "oak", Length: 46, Beam: 9, Depth: 5,
		BottomPinch: 0.15, HullFlare: 0.35, FlareCurve: 1.6,
		SheerCurve: 0.5, SheerCurveExp: 1.8, BowLength: 14, BowSharpness: 1.3,
		BowKeelRise: 0.9, BowKeelLength: 12, SternLength: 14, SternSharpness: 1.3,
		KeelCurve: 1.5, CastleBlend: 4,
		DoubleEnder: true, StemPostHeight: 4, Rocker: 0.25,
		Deadrise: 0.45, MidFullness: 0.35, BowSectionV: 0.6, StemRake: 0.7, StemCurve: -0.3,
	}
	saveHull(t, out+"/v2_longship.png", longship)
	airship := generator.HullParams{
		Version: 3, WoodType: "dark_oak", Length: 56, Beam: 15, Depth: 8,
		BottomPinch: 0.2, BowLength: 16, BowSharpness: 1.4, SternLength: 16, SternSharpness: 1.4,
		KeelCurve: 1.7, CastleBlend: 4, ClosedHull: true, DoubleEnder: true,
		MidFullness: 0.5, BowSectionV: 0.4, Rocker: 0.1, StemRake: 0.4,
	}
	saveHull(t, out+"/v2_airship.png", airship)
}
