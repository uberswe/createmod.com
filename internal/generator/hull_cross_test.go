package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestCrossFixtures(t *testing.T) {
	if os.Getenv("CROSS") == "" {
		t.Skip("set CROSS")
	}
	fixtures := []HullParams{
		{Version: 3, WoodType: "spruce", Length: 60, Beam: 13, Depth: 9, BottomPinch: 0.3, HullFlare: 0.2, FlareCurve: 2.0, Tumblehome: 0.05, TumbleCurve: 2.5, SheerCurve: 0.3, SheerCurveExp: 2.2, BowLength: 18, BowSharpness: 1.5, BowKeelRise: 0.6, BowKeelLength: 14, SternStyle: "round", SternLength: 10, SternSharpness: 0.8, SternKeelRise: 0.3, SternKeelLength: 7, KeelCurve: 1.7, CastleBlend: 4, HasRailings: true, HasTrim: true, BowStyle: "clipper", BowCurve: -0.4},
		{Version: 3, WoodType: "oak", Length: 46, Beam: 9, Depth: 5, BottomPinch: 0.15, HullFlare: 0.35, FlareCurve: 1.6, SheerCurve: 0.5, SheerCurveExp: 1.8, BowLength: 14, BowSharpness: 1.3, BowKeelRise: 0.9, BowKeelLength: 12, SternLength: 14, SternSharpness: 1.3, KeelCurve: 1.5, CastleBlend: 4, DoubleEnder: true, StemPostHeight: 4, Rocker: 0.25, Deadrise: 0.45, MidFullness: 0.35, BowSectionV: 0.6, StemRake: 0.7, StemCurve: -0.3, SternStyle: "round"},
		{Version: 3, WoodType: "dark_oak", Length: 56, Beam: 15, Depth: 8, BottomPinch: 0.2, BowLength: 16, BowSharpness: 1.4, SternLength: 16, SternSharpness: 1.4, KeelCurve: 1.7, CastleBlend: 4, ClosedHull: true, DoubleEnder: true, MidFullness: 0.5, BowSectionV: 0.4, Rocker: 0.1, StemRake: 0.4, SternStyle: "round"},
	}
	for i, p := range fixtures {
		r, err := GenerateHull(p)
		if err != nil {
			t.Fatal(err)
		}
		j, _ := json.Marshal(p)
		fmt.Printf("GO %d %s %d %s\n", i, canonicalResultHash(r), len(r.Blocks), j)
	}
}
