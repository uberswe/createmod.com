package generator

import "testing"

// The frontend (generator.js encodeCompact) drops any param equal to its
// encode-default — the page's initial slider/checkbox/select values — from the
// share hash. The decoder must therefore restore those defaults for absent
// fields; leaving them at the Go zero value makes Validate clamp them up to a
// tiny minimum, which reconstructs a different, fragmented build than the
// frontend shows.
//
// Regression: /generators/balloon/YjMuLi4uLi4uLi4uLi4uLi4x (all defaults except
// tail fins) rendered a 7x6x5 blob instead of the intended full envelope.
func TestDecodeHash_BalloonDefaultsRestored(t *testing.T) {
	res, typ, err := DecodeHash("YjMuLi4uLi4uLi4uLi4uLi4x")
	if err != nil {
		t.Fatal(err)
	}
	if typ != "balloon" {
		t.Fatalf("type = %q, want balloon", typ)
	}
	// lengthX default 36 -> a full-length envelope (~37 wide), not the clamped
	// minimum of 6 (SizeX 8) that produced the fragmented OG image.
	if res.SizeX < 34 {
		t.Errorf("SizeX = %d, want ~37 (default lengthX 36); defaults not restored", res.SizeX)
	}
	if len(res.Blocks) < 800 {
		t.Errorf("blocks = %d, want a full envelope (~1300); shape collapsed", len(res.Blocks))
	}
}

func TestDecodeBalloonParams_EmptyUsesDefaults(t *testing.T) {
	p := decodeBalloonParams(nil, 3)
	if p.LengthX != 36 || p.WidthZ != 18 || p.HeightY != 16 {
		t.Errorf("dims = %d/%d/%d, want 36/18/16", p.LengthX, p.WidthZ, p.HeightY)
	}
	if !p.Hollow || p.Shell != 1 || p.RibSpacing != 4 || p.KeelDepth != 1 || p.FinHeight != 4 || p.FinLength != 5 {
		t.Errorf("hollow=%v shell=%d ribSpacing=%d keelDepth=%d finHeight=%d finLength=%d",
			p.Hollow, p.Shell, p.RibSpacing, p.KeelDepth, p.FinHeight, p.FinLength)
	}
	if p.EnvelopeMaterial != "wool" || p.EnvelopeColor != "white" || p.FrameMaterial != "wood" || p.FrameWoodType != "spruce" {
		t.Errorf("materials = %q/%q/%q/%q", p.EnvelopeMaterial, p.EnvelopeColor, p.FrameMaterial, p.FrameWoodType)
	}
}

func TestDecodeHullParams_EmptyUsesDefaults(t *testing.T) {
	p := decodeHullParams(nil, 3)
	if p.Length != 40 || p.Beam != 10 || p.Depth != 6 {
		t.Errorf("dims = %d/%d/%d, want 40/10/6", p.Length, p.Beam, p.Depth)
	}
	if p.WoodType != "spruce" || p.SternStyle != "round" || p.BowStyle != "default" {
		t.Errorf("enums = %q/%q/%q, want spruce/round/default", p.WoodType, p.SternStyle, p.BowStyle)
	}
	if !p.HasRailings || !p.HasTrim || !p.HasWindows {
		t.Errorf("railings/trim/windows = %v/%v/%v, want all true", p.HasRailings, p.HasTrim, p.HasWindows)
	}
	if p.MidFullness != 0.65 || p.BowSectionV != 0.55 {
		t.Errorf("v2 floats midFullness=%v bowSectionV=%v, want 0.65/0.55", p.MidFullness, p.BowSectionV)
	}
}

func TestDecodePropellerParams_EmptyUsesDefaults(t *testing.T) {
	p := decodePropellerParams(nil, 2)
	if p.Blades != 4 || p.Length != 10 || p.RootChord != 3 || p.TipChord != 1 {
		t.Errorf("blades/length/root/tip = %d/%d/%d/%d, want 4/10/3/1", p.Blades, p.Length, p.RootChord, p.TipChord)
	}
	if !p.Swept || p.AirfoilShape != "curved" || p.BladeMaterial != "wool" || p.BladeColor != "white" || p.Orientation != "horizontal" {
		t.Errorf("swept=%v airfoil=%q mat=%q color=%q orient=%q", p.Swept, p.AirfoilShape, p.BladeMaterial, p.BladeColor, p.Orientation)
	}
	if p.SweepDegrees != 25 {
		t.Errorf("sweepDegrees = %v, want 25", p.SweepDegrees)
	}
}
