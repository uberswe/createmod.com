package render_test

import (
	"image/png"
	"os"
	"testing"

	"createmod/internal/generator"
	"createmod/internal/generator/render"
)

func TestPresetRenders(t *testing.T) {
	out := os.Getenv("HULL_OUT")
	if out == "" {
		t.Skip("set HULL_OUT")
	}
	presets := map[string]generator.HullParams{
		"galleon": {Version: 3, WoodType: "dark_oak", Length: 72, Beam: 22, Depth: 13, BottomPinch: 0.35, HullFlare: 0.4, FlareCurve: 2.4, Tumblehome: 0.35, TumbleCurve: 2.8, SheerCurve: 0.22, SheerCurveExp: 2, BowLength: 12, BowSharpness: 1.5, BowKeelRise: 0.7, BowKeelLength: 12, BowCurve: 0.15, SternStyle: "square", SternLength: 7, SternSharpness: 0.5, SternKeelRise: 0.4, SternKeelLength: 8, SternOverhang: 0.55, KeelCurve: 1.7, MidWidthBias: 0.2, CastleBlend: 6, HasRailings: true, HasTrim: true, HasWindows: true, CastleHeight: 4, CastleLength: 19, ForecastleHeight: 2, ForecastleLength: 9, HasGunPorts: true, GunPortRow: 3, GunPortSpacing: 5, BowStyle: "default", StemRake: 0.55, StemCurve: 0.3, SternRake: 0.2, Rocker: 0.05, Deadrise: 0.1, MidFullness: 0.7, BowSectionV: 0.5, SternFullness: 0.6, ParallelMidbody: 0.25},
		"clipper": {Version: 3, WoodType: "spruce", Length: 120, Beam: 18, Depth: 11, BottomPinch: 0.25, HullFlare: 0.22, FlareCurve: 2.6, Tumblehome: 0.26, TumbleCurve: 3, SheerCurve: 0.1, SheerCurveExp: 2, BowLength: 28, BowSharpness: 2.5, BowKeelRise: 0.4, BowKeelLength: 24, BowCurve: -0.8, SternStyle: "round", SternLength: 14, SternSharpness: 1.2, SternKeelRise: 0.2, SternKeelLength: 12, KeelCurve: 1.7, MidWidthBias: 0.3, CastleBlend: 6, HasRailings: true, HasTrim: true, HasWindows: true, CastleHeight: 2, CastleLength: 14, ForecastleHeight: 1, ForecastleLength: 8, BowStyle: "clipper", StemRake: 1.0, StemCurve: -0.7, SternRake: 0.45, Rocker: 0.05, Deadrise: 0.35, MidFullness: 0.5, BowSectionV: 0.75, SternFullness: 0.4, ParallelMidbody: 0.25},
		"dhow":    {Version: 3, WoodType: "acacia", Length: 44, Beam: 12, Depth: 7, BottomPinch: 0.35, HullFlare: 0.1, FlareCurve: 2.6, Tumblehome: 0.05, TumbleCurve: 3, SheerCurve: 0.2, SheerCurveExp: 2, BowLength: 12, BowSharpness: 2.2, BowKeelRise: 0.9, BowKeelLength: 11, BowCurve: -0.6, SternStyle: "pointed", SternLength: 7, SternSharpness: 1.5, SternKeelRise: 0.4, SternKeelLength: 7, KeelCurve: 1.7, MidWidthBias: 0.3, CastleBlend: 4, HasTrim: true, CastleHeight: 1, CastleLength: 6, BowStyle: "pointed", StemRake: 1.1, StemCurve: -0.4, SternRake: 0.5, Rocker: 0.15, Deadrise: 0.4, MidFullness: 0.4, BowSectionV: 0.65, SternFullness: 0.35, ParallelMidbody: 0.1},
	}
	for name, p := range presets {
		r, err := generator.GenerateHull(p)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		img := render.Isometric(r)
		f, _ := os.Create(out + "/preset_" + name + ".png")
		png.Encode(f, img)
		f.Close()
	}
}
