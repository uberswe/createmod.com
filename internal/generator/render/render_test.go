package render_test

import (
	"createmod/internal/generator"
	"createmod/internal/generator/render"
	"image/png"
	"os"
	"testing"
)

// A solid, convex build projects to a filled hexagon, so every horizontal span
// between its first and last painted pixel must be fully covered. The old
// renderer drew the hidden -X face instead of the visible +X face (and sorted
// by x+z instead of x+y+z), which punched saw-tooth gaps into the surface —
// the "missing/partial/inverted blocks" seen in ship previews. Guard against
// any interior background holes returning.
func TestIsometric_SolidBuildHasNoInteriorHoles(t *testing.T) {
	var bs []generator.Block
	for x := 0; x < 12; x++ {
		for z := 0; z < 12; z++ {
			for y := 0; y < 4; y++ {
				bs = append(bs, generator.Block{X: x, Y: y, Z: z, Type: generator.BlockPlank})
			}
		}
	}
	res := &generator.GenerateResult{
		Blocks: bs, SizeX: 12, SizeY: 4, SizeZ: 12,
		Materials: generator.MaterialConfig{WoodType: "spruce"},
	}
	img := render.Isometric(res)

	bg := render.BackgroundColor
	isBG := func(x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return uint8(r>>8) == bg.R && uint8(g>>8) == bg.G && uint8(b>>8) == bg.B
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	holes := 0
	for y := 0; y < h; y++ {
		first, last := -1, -1
		for x := 0; x < w; x++ {
			if !isBG(x, y) {
				if first < 0 {
					first = x
				}
				last = x
			}
		}
		for x := first + 1; x < last; x++ {
			if isBG(x, y) {
				holes++
			}
		}
	}
	// A convex solid should be gap-free; allow a tiny margin for 1px rounding at
	// the extreme tip rows.
	if holes > 15 {
		t.Errorf("solid build rendered with %d interior background holes; visible faces missing", holes)
	}
}

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

	// Verify corners are the background color.
	bg := render.BackgroundColor
	r, g, b, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != bg.R || uint8(g>>8) != bg.G || uint8(b>>8) != bg.B {
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
