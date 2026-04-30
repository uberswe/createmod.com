package generator

import (
	"testing"
)

func TestGeneratePropeller_Basic(t *testing.T) {
	result, err := GeneratePropeller(PropellerParams{
		Blades:       4,
		Length:       10,
		RootChord:    3,
		TipChord:     1,
		SweepDegrees: 25,
		Swept:        true,
		AirfoilShape: "curved",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Blocks) == 0 {
		t.Fatal("expected blocks, got none")
	}
	if result.SizeX <= 0 || result.SizeY <= 0 || result.SizeZ <= 0 {
		t.Fatalf("invalid size: %dx%dx%d", result.SizeX, result.SizeY, result.SizeZ)
	}
	for _, b := range result.Blocks {
		if b.X < 0 || b.Y < 0 || b.Z < 0 {
			t.Fatalf("negative coordinate: (%d,%d,%d)", b.X, b.Y, b.Z)
		}
	}
}

func TestGeneratePropeller_Validation(t *testing.T) {
	_, err := GeneratePropeller(PropellerParams{Blades: 1, Length: 10, RootChord: 3, TipChord: 1})
	if err == nil {
		t.Fatal("expected error for blades=1")
	}
}

func TestGenerateBalloon_Basic(t *testing.T) {
	result, err := GenerateBalloon(BalloonParams{
		LengthX:     30,
		WidthZ:      16,
		HeightY:     16,
		CylinderMid: 0,
		Hollow:      true,
		Shell:       1,
		RibSpacing:  4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Blocks) == 0 {
		t.Fatal("expected blocks, got none")
	}
}

func TestGenerateBalloon_Hollow(t *testing.T) {
	solid, _ := GenerateBalloon(BalloonParams{
		LengthX: 20, WidthZ: 20, HeightY: 20, Hollow: false, Shell: 1, RibSpacing: 4,
	})
	hollow, _ := GenerateBalloon(BalloonParams{
		LengthX: 20, WidthZ: 20, HeightY: 20, Hollow: true, Shell: 1, RibSpacing: 4,
	})
	if len(hollow.Blocks) >= len(solid.Blocks) {
		t.Fatalf("hollow (%d blocks) should have fewer blocks than solid (%d)", len(hollow.Blocks), len(solid.Blocks))
	}
}

func TestGenerateHull_Basic(t *testing.T) {
	result, err := GenerateHull(HullParams{
		Length: 40, Beam: 10, Depth: 5,
		BottomPinch: 0.35, HullFlare: 0.15, FlareCurve: 2.5,
		Tumblehome: 0.05, TumbleCurve: 3,
		SheerCurve: 0.15, SheerCurveExp: 2,
		BowLength: 8, BowSharpness: 1.2,
		SternStyle: "round", SternLength: 6, SternSharpness: 0.8,
		HasRailings: true, HasTrim: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Blocks) == 0 {
		t.Fatal("expected blocks, got none")
	}
}

func TestExportNBT(t *testing.T) {
	result, _ := GeneratePropeller(PropellerParams{
		Blades: 4, Length: 5, RootChord: 2, TipChord: 1,
		SweepDegrees: 0, Swept: false, AirfoilShape: "linear",
	})

	data, err := ExportNBT(result)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty NBT data")
	}
	// Gzip magic number
	if data[0] != 0x1f || data[1] != 0x8b {
		t.Fatal("expected gzip-compressed data")
	}
}
