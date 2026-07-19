package generator

import "testing"

// Regression for the raked-bow "teeth": lateral chamfer stairs in the
// bow/stern rake spans project full-square silhouettes below the rake line.
// Params reproduce the reported share link (long keel rises, sharp V bow).
func Test_HullV2_RakeSpansHaveNoLateralTeeth(t *testing.T) {
	p := HullParams{Version: 3, WoodType: "oak", Length: 62, Beam: 10, Depth: 5,
		BottomPinch: 0.45, HullFlare: 0.1, FlareCurve: 2.6, Tumblehome: 0.1, TumbleCurve: 3,
		SheerCurve: 0.35, SheerCurveExp: 1.8, BowLength: 16, BowSharpness: 2.2,
		BowKeelRise: 1.4, BowKeelLength: 18, BowCurve: -0.5, BowStyle: "pointed", BowSectionV: 0.55,
		SternStyle: "pointed", SternLength: 16, SternSharpness: 2, SternKeelRise: 1.3,
		SternKeelLength: 18, KeelCurve: 1.7, CastleBlend: 4, MidFullness: 0.65,
		SternFullness: 0.5, StemRake: 0.35, StemCurve: 0.15, SternRake: 0.35}
	r, err := GenerateHull(p)
	if err != nil {
		t.Fatal(err)
	}
	// In the bow/stern rake spans (last/first 12 slices of this hull), all
	// top-half chamfers must be fore-aft: lateral ones read as teeth on
	// the side silhouette.
	maxZ := 0
	for _, b := range r.Blocks {
		if b.Z > maxZ {
			maxZ = b.Z
		}
	}
	teeth := 0
	for _, b := range r.Blocks {
		if b.Type != BlockStair || b.Props["half"] != "top" {
			continue
		}
		if b.Z > 11 && b.Z < maxZ-11 {
			continue
		}
		if f := b.Props["facing"]; f == "east" || f == "west" {
			teeth++
		}
	}
	if teeth != 0 {
		t.Errorf("lateral top-half stairs in rake spans: %d", teeth)
	}
}
