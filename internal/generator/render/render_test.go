package render_test

import (
	"createmod/internal/generator"
	"createmod/internal/generator/render"
	"image/png"
	"os"
	"testing"
)

func TestIsometricRender(t *testing.T) {
	// Real hash from the generator (balloon)
	hash := "YjIuMTA0LjE1LjE2LjY1LjIwLjIwLjAuMC4xLjEuMS42LjAuMS4xLjAuNC42LmUuZ3kuYS5z"

	result, genType, err := generator.DecodeHash(hash)
	if err != nil {
		t.Fatalf("DecodeHash error: %v", err)
	}

	if genType != "balloon" {
		t.Errorf("expected balloon, got %s", genType)
	}

	if len(result.Blocks) == 0 {
		t.Fatal("no blocks generated")
	}

	t.Logf("Type: %s, Blocks: %d, Size: %dx%dx%d", genType, len(result.Blocks), result.SizeX, result.SizeY, result.SizeZ)

	img := render.Isometric(result)
	if img.Rect.Dx() != 800 || img.Rect.Dy() != 450 {
		t.Errorf("unexpected image size: %dx%d", img.Rect.Dx(), img.Rect.Dy())
	}

	// Verify corners are the background color (#3a7098)
	r, g, b, _ := img.At(0, 0).RGBA()
	if r>>8 != 58 || g>>8 != 112 || b>>8 != 152 {
		t.Errorf("top-left corner not background color: got #%02x%02x%02x", r>>8, g>>8, b>>8)
	}

	// Save for visual inspection
	f, err := os.Create("/tmp/test_preview.png")
	if err == nil {
		defer f.Close()
		_ = png.Encode(f, img)
		t.Log("Saved preview to /tmp/test_preview.png")
	}
}

func TestIsometricHull(t *testing.T) {
	hash := "aDIuY2guMTA1LjMzLjEzLjQ1LjAuNDAwLjMwLjMwMC4wLjI1MC4zMS4xNzAuMTIwLjQwLnIuMzAuNzAuNzAuMzAuMjAwLjUuMC4wLjAuMy4xMi4yLjYuMC4yLjQuNTAuNjAuMA"

	result, genType, err := generator.DecodeHash(hash)
	if err != nil {
		t.Fatalf("DecodeHash error: %v", err)
	}

	if genType != "hull" {
		t.Errorf("expected hull, got %s", genType)
	}

	t.Logf("Type: %s, Blocks: %d, Size: %dx%dx%d", genType, len(result.Blocks), result.SizeX, result.SizeY, result.SizeZ)

	img := render.Isometric(result)

	f, err := os.Create("/tmp/test_hull.png")
	if err == nil {
		defer f.Close()
		_ = png.Encode(f, img)
		t.Log("Saved hull preview to /tmp/test_hull.png")
	}
}

func TestIsometricPropeller(t *testing.T) {
	// Propeller hash
	hash := "cDIuNC4xNS4xNi4xMC42NS4xLmwudy53aA"

	result, genType, err := generator.DecodeHash(hash)
	if err != nil {
		t.Fatalf("DecodeHash error: %v", err)
	}

	if genType != "propeller" {
		t.Errorf("expected propeller, got %s", genType)
	}

	t.Logf("Type: %s, Blocks: %d, Size: %dx%dx%d", genType, len(result.Blocks), result.SizeX, result.SizeY, result.SizeZ)

	img := render.Isometric(result)
	if img.Rect.Dx() != 800 || img.Rect.Dy() != 450 {
		t.Errorf("unexpected image size: %dx%d", img.Rect.Dx(), img.Rect.Dy())
	}

	f, err := os.Create("/tmp/test_propeller.png")
	if err == nil {
		defer f.Close()
		_ = png.Encode(f, img)
		t.Log("Saved propeller preview to /tmp/test_propeller.png")
	}
}
