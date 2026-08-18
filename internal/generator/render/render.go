// Package render provides an isometric 2D renderer for generator block models.
package render

import (
	"image"
	"image/color"
	"math"
	"sort"

	"createmod/internal/generator"
)

const (
	imgWidth  = 800
	imgHeight = 450
)

// BackgroundColor is the dark-mode canvas the previews render on (#1f2121),
// matching the site's dark surface so shared images sit on brand.
var BackgroundColor = color.RGBA{R: 0x1f, G: 0x21, B: 0x21, A: 255}

// Isometric renders a GenerateResult as an isometric PNG image.
func Isometric(result *generator.GenerateResult) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Fill background
	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			img.SetRGBA(x, y, BackgroundColor)
		}
	}

	if len(result.Blocks) == 0 {
		return img
	}

	// Filter out air blocks and build sorted list
	blocks := make([]generator.Block, 0, len(result.Blocks))
	for _, b := range result.Blocks {
		if b.Type != generator.BlockAir {
			blocks = append(blocks, b)
		}
	}

	if len(blocks) == 0 {
		return img
	}

	// Sort for painter's algorithm: back-to-front.
	// The camera at (+X, +Y, +Z) looks along (-1, -1, -1), so a block's depth is
	// x+y+z: larger sums are closer to the camera and must be drawn last. Sorting
	// by (x+z) then y is wrong — a tall, near block (high y) would be drawn before
	// a lower, farther block, letting the farther block overdraw it, which reads
	// as missing/partial/inverted blocks in the preview. Blocks sharing a depth
	// plane cannot occlude one another, so their order is only for determinism.
	sort.Slice(blocks, func(i, j int) bool {
		di := blocks[i].X + blocks[i].Y + blocks[i].Z
		dj := blocks[j].X + blocks[j].Y + blocks[j].Z
		if di != dj {
			return di < dj
		}
		if blocks[i].Y != blocks[j].Y {
			return blocks[i].Y < blocks[j].Y
		}
		return blocks[i].X < blocks[j].X
	})

	// Compute model center
	cx := float64(result.SizeX) / 2.0
	cy := float64(result.SizeY) / 2.0
	cz := float64(result.SizeZ) / 2.0

	// Determine scale to fit model in image with padding
	maxExtent := math.Max(float64(result.SizeX), math.Max(float64(result.SizeY), float64(result.SizeZ)))
	if maxExtent == 0 {
		maxExtent = 1
	}
	scale := float64(imgHeight) * 0.7 / maxExtent

	// Isometric projection constants (2:1 isometric)
	// Screen X = (x - z) * cos30 * scale
	// Screen Y = (x + z) * sin30 * scale - y * scale
	cos30 := math.Cos(math.Pi / 6) // ~0.866
	sin30 := 0.5

	// Project center to find offset
	centerSX := (cx - cz) * cos30 * scale
	centerSY := (cx+cz)*sin30*scale - cy*scale
	offsetX := float64(imgWidth)/2 - centerSX
	offsetY := float64(imgHeight)/2 - centerSY

	// Project function: world → screen
	project := func(wx, wy, wz float64) (int, int) {
		sx := (wx-wz)*cos30*scale + offsetX
		sy := (wx+wz)*sin30*scale - wy*scale + offsetY
		return int(math.Round(sx)), int(math.Round(sy))
	}

	// Draw each block
	for _, b := range blocks {
		bx := float64(b.X)
		by := float64(b.Y)
		bz := float64(b.Z)

		baseColor := blockColor(b, result.Materials)

		// Block height (slabs are half)
		height := 1.0
		yOff := 0.0
		if b.Type == generator.BlockSlabBot {
			height = 0.5
		} else if b.Type == generator.BlockSlabTop {
			height = 0.5
			yOff = 0.5
		} else if b.Type == generator.BlockFence {
			drawFenceBlock(img, project, bx, by, bz, baseColor)
			continue
		} else if b.Type == generator.BlockStair {
			// Stairs were previously drawn as full cubes, which turned the
			// chamfered hull surface into a blocky wall in the preview. Draw the
			// actual two-box stair so the render matches the frontend's stepped
			// silhouette. Facing comes from the generated (vanilla-semantics)
			// props, same as the frontend and the export.
			drawStair(img, project, bx, by, bz, b.Props["facing"], b.Props["half"], baseColor)
			continue
		} else if b.Type == generator.BlockTrapdoor {
			height = 0.2
		}

		drawCube(img, project, bx, by+yOff, bz, 1.0, height, 1.0, baseColor)
	}

	return img
}

// drawStair renders a straight stair as a full half-height slab plus a
// half-footprint step box on the facing side (the vanilla oak_stairs shape:
// facing=east puts the tall part on +X). half=top flips which half is the full
// slab. The two boxes are drawn back-to-front for the +X/+Z isometric camera.
func drawStair(img *image.RGBA, project func(float64, float64, float64) (int, int),
	bx, by, bz float64, facing, half string, base color.RGBA) {
	slabY, stepY := by, by+0.5
	if half == "top" {
		slabY, stepY = by+0.5, by
	}
	// Step box occupies the facing half of the footprint.
	sx, sz, sw, sd := bx, bz, 1.0, 1.0
	switch facing {
	case "east":
		sx, sw = bx+0.5, 0.5
	case "west":
		sw = 0.5
	case "south":
		sz, sd = bz+0.5, 0.5
	case "north":
		sd = 0.5
	}
	drawSlab := func() { drawCube(img, project, bx, slabY, bz, 1.0, 0.5, 1.0, base) }
	drawStep := func() { drawCube(img, project, sx, stepY, sz, sw, 0.5, sd, base) }
	// The two sub-boxes are stacked, so the upper one is closer to the
	// (+X,+Y,+Z) camera and must be drawn last (painter's order by height, the
	// same x+y+z rule as whole blocks). Ordering by facing instead let the lower
	// box overdraw the upper one for half=top / far-facing stairs, erasing the
	// inside face and leaving a hollow-looking top.
	if half == "top" { // slab is the upper box
		drawStep()
		drawSlab()
	} else { // step is the upper box
		drawSlab()
		drawStep()
	}
}

// drawCube draws a cube with three visible faces (top, left, right) using the painter's algorithm.
func drawCube(img *image.RGBA, project func(float64, float64, float64) (int, int),
	x, y, z, w, h, d float64, base color.RGBA) {

	topColor := lighten(base, 1.2)
	leftColor := darken(base, 0.7)
	rightColor := darken(base, 0.85)

	// 8 corners of the cube
	// Top face: 4 corners at y+h
	t0x, t0y := project(x, y+h, z)
	t1x, t1y := project(x+w, y+h, z)
	t2x, t2y := project(x+w, y+h, z+d)
	t3x, t3y := project(x, y+h, z+d)

	// Bottom face: 4 corners at y
	b0x, b0y := project(x, y, z)
	b1x, b1y := project(x+w, y, z)
	b2x, b2y := project(x+w, y, z+d)
	b3x, b3y := project(x, y, z+d)

	// The camera at (+X,+Y,+Z) sees exactly three cube faces: +X, +Z and the top
	// (+Y). Drawing the x=0 (-X) back face instead of the +X face left every
	// block missing its right side, which shows up as saw-tooth gaps along Z-runs
	// and vertical columns.
	//
	// +X face (x+w plane), the right-facing side: t1, t2, b2, b1
	fillQuad(img, t1x, t1y, t2x, t2y, b2x, b2y, b1x, b1y, rightColor)

	// +Z face (z+d plane), the left-facing side: t3, t2, b2, b3
	fillQuad(img, t3x, t3y, t2x, t2y, b2x, b2y, b3x, b3y, leftColor)

	// Top face (+Y): t0, t1, t2, t3
	fillQuad(img, t0x, t0y, t1x, t1y, t2x, t2y, t3x, t3y, topColor)

	// Outline the visible edges so adjacent same-colored blocks stay distinct,
	// mirroring the frontend's per-block edges (EdgesGeometry). Only the seams of
	// the three visible faces are drawn; the painter's order keeps nearer blocks
	// overdrawing farther ones so hidden seams never show.
	edge := darken(base, 0.5)
	// Top face rim.
	drawLine(img, t0x, t0y, t1x, t1y, edge)
	drawLine(img, t1x, t1y, t2x, t2y, edge)
	drawLine(img, t2x, t2y, t3x, t3y, edge)
	drawLine(img, t3x, t3y, t0x, t0y, edge)
	// The three vertical seams down from the front corners.
	drawLine(img, t1x, t1y, b1x, b1y, edge)
	drawLine(img, t2x, t2y, b2x, b2y, edge)
	drawLine(img, t3x, t3y, b3x, b3y, edge)
	// Bottom of the two side faces.
	drawLine(img, b1x, b1y, b2x, b2y, edge)
	drawLine(img, b2x, b2y, b3x, b3y, edge)

	_ = b0x
	_ = b0y
}

// drawLine draws a 1px line with Bresenham's algorithm, clipped to the image.
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		if x0 >= 0 && x0 < imgWidth && y0 >= 0 && y0 < imgHeight {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawFenceBlock(img *image.RGBA, project func(float64, float64, float64) (int, int),
	x, y, z float64, base color.RGBA) {
	// Draw fence as a thin post
	postW := 0.25
	postOff := 0.375
	drawCube(img, project, x+postOff, y, z+postOff, postW, 1.0, postW, base)
}

// fillQuad fills a quadrilateral defined by 4 points with a solid color using scanline.
func fillQuad(img *image.RGBA, x0, y0, x1, y1, x2, y2, x3, y3 int, c color.RGBA) {
	// Find bounding box
	minY := min4(y0, y1, y2, y3)
	maxY := max4(y0, y1, y2, y3)
	minX := min4(x0, x1, x2, x3)
	maxX := max4(x0, x1, x2, x3)

	// Clip to image bounds
	if minY < 0 {
		minY = 0
	}
	if maxY >= imgHeight {
		maxY = imgHeight - 1
	}
	if minX < 0 {
		minX = 0
	}
	if maxX >= imgWidth {
		maxX = imgWidth - 1
	}

	// Edges of the quad
	edges := [4][4]int{
		{x0, y0, x1, y1},
		{x1, y1, x2, y2},
		{x2, y2, x3, y3},
		{x3, y3, x0, y0},
	}

	// Scanline fill
	for sy := minY; sy <= maxY; sy++ {
		xMin := maxX + 1
		xMax := minX - 1

		for _, e := range edges {
			ex0, ey0, ex1, ey1 := e[0], e[1], e[2], e[3]
			if (ey0 <= sy && ey1 > sy) || (ey1 <= sy && ey0 > sy) {
				// Edge crosses this scanline
				t := float64(sy-ey0) / float64(ey1-ey0)
				ix := int(math.Round(float64(ex0) + t*float64(ex1-ex0)))
				if ix < xMin {
					xMin = ix
				}
				if ix > xMax {
					xMax = ix
				}
			}
		}

		if xMin > xMax {
			continue
		}
		if xMin < 0 {
			xMin = 0
		}
		if xMax >= imgWidth {
			xMax = imgWidth - 1
		}

		for sx := xMin; sx <= xMax; sx++ {
			img.SetRGBA(sx, sy, c)
		}
	}
}

func lighten(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: clampU8(float64(c.R) * factor),
		G: clampU8(float64(c.G) * factor),
		B: clampU8(float64(c.B) * factor),
		A: 255,
	}
}

func darken(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: clampU8(float64(c.R) * factor),
		G: clampU8(float64(c.G) * factor),
		B: clampU8(float64(c.B) * factor),
		A: 255,
	}
}

func clampU8(v float64) uint8 {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return uint8(v)
}

func min4(a, b, c, d int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}

func max4(a, b, c, d int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	if d > m {
		m = d
	}
	return m
}
