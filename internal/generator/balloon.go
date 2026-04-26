package generator

import (
	"fmt"
	"math"
)

type BalloonParams struct {
	LengthX       int     `json:"lengthX"`
	WidthZ        int     `json:"widthZ"`
	HeightY       int     `json:"heightY"`
	CylinderMid   float64 `json:"cylinderMid"`
	FrontTaper    float64 `json:"frontTaper"`
	RearTaper     float64 `json:"rearTaper"`
	TopFlatten    float64 `json:"topFlatten"`
	BottomFlatten float64 `json:"bottomFlatten"`
	Shell         int     `json:"shell"`
	Hollow        bool    `json:"hollow"`
	RibEnabled    bool    `json:"ribEnabled"`
	RibSpacing    int     `json:"ribSpacing"`
	KeelEnabled   bool    `json:"keelEnabled"`
	KeelDepth     int     `json:"keelDepth"`
	FinEnabled    bool    `json:"finEnabled"`
	FinHeight     int     `json:"finHeight"`
	FinLength     int     `json:"finLength"`
}

func (p *BalloonParams) Validate() error {
	clampInt := func(v *int, lo, hi int) {
		if *v < lo {
			*v = lo
		}
		if *v > hi {
			*v = hi
		}
	}
	clampFloat := func(v *float64, lo, hi float64) {
		if *v < lo {
			*v = lo
		}
		if *v > hi {
			*v = hi
		}
	}

	clampInt(&p.LengthX, 6, 120)
	clampInt(&p.WidthZ, 4, 60)
	clampInt(&p.HeightY, 4, 60)
	clampFloat(&p.CylinderMid, 0, 0.85)
	clampFloat(&p.FrontTaper, 0, 1)
	clampFloat(&p.RearTaper, 0, 1)
	clampFloat(&p.TopFlatten, 0, 0.5)
	clampFloat(&p.BottomFlatten, 0, 0.5)
	clampInt(&p.Shell, 1, 5)
	clampInt(&p.RibSpacing, 2, 12)
	clampInt(&p.KeelDepth, 0, 10)
	clampInt(&p.FinHeight, 2, 15)
	clampInt(&p.FinLength, 3, 20)

	if p.LengthX < 6 {
		return fmt.Errorf("lengthX must be at least 6")
	}
	return nil
}

func GenerateBalloon(p BalloonParams) (*GenerateResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	rX := float64(p.LengthX) / 2
	rY := float64(p.HeightY) / 2
	rZ := float64(p.WidthZ) / 2

	midHalf := rX * p.CylinderMid
	capLen := math.Max(1, rX-midHalf)

	cx := math.Floor(rX)
	cy := math.Floor(rY)
	cz := math.Floor(rZ)

	sizeX := p.LengthX + 1
	sizeY := p.HeightY + 1
	sizeZ := p.WidthZ + 1

	type voxel struct {
		blockType int
	}

	grid := make(map[[3]int]*voxel)

	eDist := func(x, y, z int) float64 {
		dx := float64(x) - cx
		dy := float64(y) - cy
		dz := float64(z) - cz

		var nx float64
		if math.Abs(dx) <= midHalf {
			nx = 0
		} else {
			if dx > 0 {
				nx = (dx - midHalf) / capLen
			} else {
				nx = (dx + midHalf) / capLen
			}
		}

		ny := dy / rY
		nz := dz / rZ

		// Taper
		if math.Abs(dx) > midHalf {
			var taper float64
			var t float64
			if dx > 0 {
				t = (dx - midHalf) / capLen
				taper = p.FrontTaper
			} else {
				t = (-dx - midHalf) / capLen
				taper = p.RearTaper
			}
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			scale := 1 + taper*t*t*3
			ny *= scale
			nz *= scale
		}

		// Flatten
		if dy > 0 && p.TopFlatten > 0 {
			ny *= 1 + p.TopFlatten
		}
		if dy < 0 && p.BottomFlatten > 0 {
			ny *= 1 + p.BottomFlatten
		}

		return nx*nx + ny*ny + nz*nz
	}

	// Pass 1: collect interior blocks
	for x := 0; x < sizeX; x++ {
		for y := 0; y < sizeY; y++ {
			for z := 0; z < sizeZ; z++ {
				if eDist(x, y, z) <= 1.0 {
					grid[[3]int{x, y, z}] = &voxel{blockType: BlockPlank}
				}
			}
		}
	}

	if p.Hollow {
		// Pass 1b: find surface blocks (have at least one air neighbor)
		surface := make(map[[3]int]bool)
		dirs := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
		for pos := range grid {
			for _, d := range dirs {
				nb := [3]int{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
				if grid[nb] == nil {
					surface[pos] = true
					break
				}
			}
		}

		// Pass 1c: flood fill inward for shell thickness
		shell := make(map[[3]int]bool)
		for pos := range surface {
			shell[pos] = true
		}
		for layer := 1; layer < p.Shell; layer++ {
			nextLayer := make(map[[3]int]bool)
			for pos := range shell {
				for _, d := range dirs {
					nb := [3]int{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
					if grid[nb] != nil && !shell[nb] {
						nextLayer[nb] = true
					}
				}
			}
			for pos := range nextLayer {
				shell[pos] = true
			}
		}

		// Remove interior blocks not in shell
		for pos := range grid {
			if !shell[pos] {
				delete(grid, pos)
			}
		}
	}

	// Pass 2: smoothing/detailing — replace surface planks with slabs for rounder edges
	{
		dirs := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
		type slabChange struct {
			pos       [3]int
			blockType int
		}
		var changes []slabChange
		for pos, v := range grid {
			if v.blockType != BlockPlank {
				continue
			}
			neighborCount := 0
			hasAbove := false
			hasBelow := false
			for _, d := range dirs {
				nb := [3]int{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
				if grid[nb] != nil {
					neighborCount++
				}
			}
			if grid[[3]int{pos[0], pos[1] + 1, pos[2]}] != nil {
				hasAbove = true
			}
			if grid[[3]int{pos[0], pos[1] - 1, pos[2]}] != nil {
				hasBelow = true
			}
			if neighborCount <= 4 {
				if hasBelow && !hasAbove {
					changes = append(changes, slabChange{pos, BlockSlabTop})
				} else if hasAbove && !hasBelow {
					changes = append(changes, slabChange{pos, BlockSlabBot})
				}
			}
		}
		for _, c := range changes {
			grid[c.pos].blockType = c.blockType
		}
	}

	// Pass 3: ribbing
	if p.RibEnabled && p.RibSpacing > 0 {
		for pos, v := range grid {
			if v.blockType == BlockPlank && pos[0]%p.RibSpacing == 0 {
				v.blockType = BlockLog
			}
		}
	}

	// Pass 4: keel
	if p.KeelEnabled && p.KeelDepth > 0 {
		minY := sizeY
		for pos := range grid {
			if pos[1] < minY {
				minY = pos[1]
			}
		}
		for x := 0; x < sizeX; x++ {
			// Find center Z column
			midZ := int(cz)
			if grid[[3]int{x, midZ, minY}] != nil || grid[[3]int{x, midZ, minY + 1}] != nil {
				for dy := 1; dy <= p.KeelDepth; dy++ {
					ky := minY - dy
					if ky >= 0 {
						grid[[3]int{x, ky, midZ}] = &voxel{blockType: BlockLog}
						sizeY = max(sizeY, ky+1)
					}
				}
			}
		}
	}

	// Pass 5: tail fins
	if p.FinEnabled && p.FinHeight > 0 && p.FinLength > 0 {
		// Find the extents needed for fin placement
		maxX := 0
		minY := sizeY
		midZ := int(cz)
		for pos := range grid {
			if pos[0] > maxX {
				maxX = pos[0]
			}
			if pos[1] < minY {
				minY = pos[1]
			}
		}

		startX := maxX - p.FinLength
		if startX < 0 {
			startX = 0
		}
		for dx := 0; dx <= p.FinLength; dx++ {
			x := startX + dx
			t := float64(dx) / float64(p.FinLength)
			height := int(float64(p.FinHeight) * (1.0 - t*t))
			if height < 1 {
				continue
			}
			for dy := 1; dy <= height; dy++ {
				ky := minY - dy
				grid[[3]int{x, ky, midZ}] = &voxel{blockType: BlockWool}
			}
		}
	}

	// Build result — normalize coordinates so all are non-negative
	minGridY := 0
	for pos := range grid {
		if pos[1] < minGridY {
			minGridY = pos[1]
		}
	}
	yOffset := 0
	if minGridY < 0 {
		yOffset = -minGridY
	}

	var blocks []Block
	actualMaxX, actualMaxY, actualMaxZ := 0, 0, 0
	for pos, v := range grid {
		ny := pos[1] + yOffset
		blocks = append(blocks, Block{X: pos[0], Y: ny, Z: pos[2], Type: v.blockType})
		if pos[0] > actualMaxX {
			actualMaxX = pos[0]
		}
		if ny > actualMaxY {
			actualMaxY = ny
		}
		if pos[2] > actualMaxZ {
			actualMaxZ = pos[2]
		}
	}

	return &GenerateResult{
		Blocks: blocks,
		SizeX:  actualMaxX + 1,
		SizeY:  actualMaxY + 1,
		SizeZ:  actualMaxZ + 1,
	}, nil
}

