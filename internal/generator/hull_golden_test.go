package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// canonicalResultHash produces an order-independent fingerprint of a
// generation result: every block with sorted props, sorted, hashed.
func canonicalResultHash(r *GenerateResult) string {
	lines := make([]string, 0, len(r.Blocks))
	for _, b := range r.Blocks {
		props := make([]string, 0, len(b.Props))
		for k, v := range b.Props {
			props = append(props, k+"="+v)
		}
		sort.Strings(props)
		lines = append(lines, fmt.Sprintf("%d,%d,%d,%d,%s", b.X, b.Y, b.Z, b.Type, strings.Join(props, ";")))
	}
	sort.Strings(lines)
	header := fmt.Sprintf("%dx%dx%d\n", r.SizeX, r.SizeY, r.SizeZ)
	sum := sha256.Sum256([]byte(header + strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:12])
}

// goldenHullV1Fixtures freeze the v1 algorithm: every pre-v3 share link
// regenerates through generateHullV1 and must reproduce identical output
// forever. If one of these hashes changes, v1 changed — revert the change.
var goldenHullV1Fixtures = []struct {
	name   string
	params HullParams
	want   string
}{
	{
		name: "sloop_defaults",
		params: HullParams{
			Version: 2, WoodType: "spruce", Length: 40, Beam: 11, Depth: 8,
			BottomPinch: 0.35, HullFlare: 0.15, FlareCurve: 2.0, Tumblehome: 0.1, TumbleCurve: 2.5,
			SheerCurve: 0.25, SheerCurveExp: 2.0, BowLength: 12, BowSharpness: 1.4,
			BowKeelRise: 0.5, BowKeelLength: 10, SternStyle: "round", SternLength: 8,
			SternSharpness: 0.8, SternKeelRise: 0.3, SternKeelLength: 6, KeelCurve: 1.7,
			CastleBlend: 4, HasRailings: true, HasTrim: true,
		},
		want: "d2e7b252c84cced893358921",
	},
	{
		name: "galleon_castles_gunports",
		params: HullParams{
			Version: 2, WoodType: "dark_oak", Length: 80, Beam: 19, Depth: 12,
			BottomPinch: 0.4, HullFlare: 0.2, FlareCurve: 1.8, Tumblehome: 0.25, TumbleCurve: 2.0,
			SheerCurve: 0.4, SheerCurveExp: 2.2, BowLength: 18, BowSharpness: 1.2,
			BowKeelRise: 0.6, BowKeelLength: 14, SternStyle: "square", SternLength: 12,
			SternSharpness: 0.6, SternKeelRise: 0.2, SternKeelLength: 8, KeelCurve: 1.7,
			CastleBlend: 6, CastleHeight: 4, CastleLength: 22, ForecastleHeight: 2, ForecastleLength: 12,
			HasRailings: true, HasTrim: true, HasWindows: true,
			HasGunPorts: true, GunPortRow: 2, GunPortSpacing: 4,
			SternOverhang: 0.4, BowStyle: "default",
		},
		want: "0b2dfc2366239dafc2a44fc8",
	},
	{
		name: "clipper_bow_v1_hash",
		params: HullParams{
			Version: 1, WoodType: "oak", Length: 60, Beam: 13, Depth: 9,
			BottomPinch: 0.3, HullFlare: 0.3, FlareCurve: 2.2, SheerCurve: 0.3, SheerCurveExp: 2.5,
			BowLength: 20, BowSharpness: 2.0, BowCurve: -0.5, BowStyle: "clipper",
			BowKeelRise: 0.8, BowKeelLength: 16, SternStyle: "pointed", SternLength: 10,
			SternSharpness: 1.0, KeelCurve: 2.0, CastleBlend: 4, MidWidthBias: 0.4,
		},
		want: "43a372e845276a8692ec9ce1",
	},
	{
		name: "minimal_barge",
		params: HullParams{
			Version: 2, WoodType: "birch", Length: 24, Beam: 8, Depth: 5,
			BottomPinch: 0.6, SternStyle: "square", BowLength: 4, SternLength: 3,
			BowSharpness: 0.8, SternSharpness: 0.5, KeelCurve: 1.7, CastleBlend: 4,
		},
		want: "6f21e343b6b35c2715459494",
	},
}

func Test_GenerateHullV1_Golden(t *testing.T) {
	for _, fx := range goldenHullV1Fixtures {
		r, err := GenerateHull(fx.params)
		if err != nil {
			t.Fatalf("%s: %v", fx.name, err)
		}
		got := canonicalResultHash(r)
		if fx.want == "" {
			t.Errorf("%s: fixture hash not set; current = %q", fx.name, got)
			continue
		}
		if got != fx.want {
			t.Errorf("%s: v1 output changed! got %s want %s — v1 must stay frozen", fx.name, got, fx.want)
		}
	}
}

// Version routing: <=2 goes to v1, >=3 goes to v2.
func Test_GenerateHull_VersionDispatch(t *testing.T) {
	base := goldenHullV1Fixtures[0].params

	v1 := base
	v1.Version = 2
	r1, err := GenerateHull(v1)
	if err != nil {
		t.Fatal(err)
	}

	v0 := base
	v0.Version = 1
	r0, err := GenerateHull(v0)
	if err != nil {
		t.Fatal(err)
	}
	if canonicalResultHash(r0) != canonicalResultHash(r1) {
		t.Errorf("versions 1 and 2 must run the same algorithm")
	}
}
