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

var bgColor = color.RGBA{R: 58, G: 112, B: 152, A: 255} // #3a7098

// Isometric renders a GenerateResult as an isometric PNG image.
func Isometric(result *generator.GenerateResult) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Fill background
	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			img.SetRGBA(x, y, bgColor)
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

	// Sort for painter's algorithm: back-to-front
	// In isometric with camera at (+X, +Y, +Z) looking toward origin:
	// draw blocks with smaller (x+z) first, then lower y first
	sort.Slice(blocks, func(i, j int) bool {
		si := blocks[i].X + blocks[i].Z
		sj := blocks[j].X + blocks[j].Z
		if si != sj {
			return si < sj
		}
		return blocks[i].Y < blocks[j].Y
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
		} else if b.Type == generator.BlockTrapdoor {
			height = 0.2
		}

		drawCube(img, project, bx, by+yOff, bz, 1.0, height, 1.0, baseColor)
	}

	return img
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

	// Draw left face (x=0 side, visible from left): t0, t3, b3, b0
	fillQuad(img, t0x, t0y, t3x, t3y, b3x, b3y, b0x, b0y, leftColor)

	// Draw right face (z+d side, visible from right): t3, t2, b2, b3
	fillQuad(img, t3x, t3y, t2x, t2y, b2x, b2y, b3x, b3y, rightColor)

	// Draw top face: t0, t1, t2, t3
	fillQuad(img, t0x, t0y, t1x, t1y, t2x, t2y, t3x, t3y, topColor)

	_ = b1x
	_ = b1y
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
