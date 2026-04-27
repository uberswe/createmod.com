package generator

import (
	"fmt"
	"math"
)

type BalloonParams struct {
	Version          int     `json:"version"`
	LengthX          int     `json:"lengthX"`
	WidthZ           int     `json:"widthZ"`
	HeightY          int     `json:"heightY"`
	CylinderMid      float64 `json:"cylinderMid"`
	FrontTaper       float64 `json:"frontTaper"`
	RearTaper        float64 `json:"rearTaper"`
	TopFlatten       float64 `json:"topFlatten"`
	BottomFlatten    float64 `json:"bottomFlatten"`
	Shell            int     `json:"shell"`
	Hollow           bool    `json:"hollow"`
	RibEnabled       bool    `json:"ribEnabled"`
	RibSpacing       int     `json:"ribSpacing"`
	KeelEnabled      bool    `json:"keelEnabled"`
	KeelDepth        int     `json:"keelDepth"`
	FinEnabled       bool    `json:"finEnabled"`
	SideFinEnabled   bool    `json:"sideFinEnabled"`
	FinHeight        int     `json:"finHeight"`
	FinLength        int     `json:"finLength"`
	EnvelopeMaterial string  `json:"envelopeMaterial"`
	EnvelopeColor    string  `json:"envelopeColor"`
	FrameMaterial    string  `json:"frameMaterial"`
	FrameWoodType    string  `json:"frameWoodType"`
}

func (p *BalloonParams) Validate() error {
	if p.Version == 0 {
		p.Version = CurrentVersion
	}
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
	if p.EnvelopeMaterial != "wool" && p.EnvelopeMaterial != "envelope" {
		p.EnvelopeMaterial = "wool"
	}
	if !isValidWoolColor(p.EnvelopeColor) {
		p.EnvelopeColor = "white"
	}
	if p.FrameMaterial != "wood" && p.FrameMaterial != "andesite_casing" {
		p.FrameMaterial = "wood"
	}
	if !isValidWoodType(p.FrameWoodType) {
		p.FrameWoodType = "spruce"
	}
	return nil
}

type coord [3]int

func coordKey(x, y, z int) coord {
	return coord{x, y, z}
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

	cX := math.Floor(rX)
	cY := math.Floor(rY)
	cZ := math.Floor(rZ)

	sizeX := p.LengthX + 2
	sizeY := p.HeightY + 2
	sizeZ := p.WidthZ + 2

	// Ellipsoid distance function matching the reference exactly.
	// In the reference, Y axis: ny < 0 means "above center" (lower y values),
	// ny > 0 means "below center" (higher y values).
	// topFlatten applies when ny < 0 (above center, lower y),
	// bottomFlatten applies when ny > 0 (below center, higher y).
	eDist := func(x, y, z int) float64 {
		ax := float64(x) - cX

		var nx float64
		if midHalf > 0 && math.Abs(ax) <= midHalf {
			nx = 0
		} else {
			capOffset := math.Abs(ax)
			if midHalf > 0 {
				capOffset -= midHalf
			}
			nx = capOffset / capLen
			if ax < 0 {
				nx = -nx
			}
		}

		ny := (float64(y) - cY) / rY
		nz := (float64(z) - cZ) / rZ

		// Taper — only in cap region
		if p.FrontTaper > 0 && nx < 0 {
			t := math.Abs(nx)
			sq := 1 + p.FrontTaper*t*t*3
			ny *= sq
			nz *= sq
		}
		if p.RearTaper > 0 && nx > 0 {
			t := math.Abs(nx)
			sq := 1 + p.RearTaper*t*t*3
			ny *= sq
			nz *= sq
		}

		// Flatten — reference: topFlatten when ny < 0, bottomFlatten when ny > 0
		// Uses (1 - flatten * 0.5) to DECREASE ny, making the shape wider/flatter
		if p.TopFlatten > 0 && ny < 0 {
			ny *= (1 - p.TopFlatten*0.5)
		}
		if p.BottomFlatten > 0 && ny > 0 {
			ny *= (1 - p.BottomFlatten*0.5)
		}

		return nx*nx + ny*ny + nz*nz
	}

	dirs := [6]coord{
		{1, 0, 0}, {-1, 0, 0},
		{0, 1, 0}, {0, -1, 0},
		{0, 0, 1}, {0, 0, -1},
	}

	type voxel struct {
		blockType int
		props     map[string]string
	}

	grid := make(map[coord]*voxel)

	// Pass 1a: Collect all blocks inside the outer ellipsoid
	insideSet := make(map[coord]bool)
	for x := 0; x < sizeX; x++ {
		for y := 0; y < sizeY; y++ {
			for z := 0; z < sizeZ; z++ {
				if eDist(x, y, z) <= 1.0 {
					insideSet[coordKey(x, y, z)] = true
				}
			}
		}
	}

	if !p.Hollow {
		// Solid fill
		for pos := range insideSet {
			grid[pos] = &voxel{blockType: BlockWool}
		}
	} else {
		// Pass 1b: Surface layer — block is on surface if at least one neighbor is outside
		shellSet := make(map[coord]bool)
		for pos := range insideSet {
			for _, d := range dirs {
				nb := coordKey(pos[0]+d[0], pos[1]+d[1], pos[2]+d[2])
				if !insideSet[nb] {
					shellSet[pos] = true
					break
				}
			}
		}

		// Pass 1c: Thicken shell inward for shell > 1
		for layer := 1; layer < p.Shell; layer++ {
			var newLayer []coord
			for pos := range insideSet {
				if shellSet[pos] {
					continue
				}
				for _, d := range dirs {
					nb := coordKey(pos[0]+d[0], pos[1]+d[1], pos[2]+d[2])
					if shellSet[nb] {
						newLayer = append(newLayer, pos)
						break
					}
				}
			}
			for _, pos := range newLayer {
				shellSet[pos] = true
			}
		}

		// Place shell blocks
		for pos := range shellSet {
			grid[pos] = &voxel{blockType: BlockWool}
		}
	}

	// Pass 2: Smoothing with slabs and stairs
	{
		type blockChange struct {
			pos coord
			v   *voxel
		}
		var changes []blockChange

		for pos, v := range grid {
			if v.blockType != BlockWool {
				continue
			}

			above := grid[coordKey(pos[0], pos[1]-1, pos[2])] != nil
			below := grid[coordKey(pos[0], pos[1]+1, pos[2])] != nil
			north := grid[coordKey(pos[0], pos[1], pos[2]-1)] != nil
			south := grid[coordKey(pos[0], pos[1], pos[2]+1)] != nil
			east := grid[coordKey(pos[0]+1, pos[1], pos[2])] != nil
			west := grid[coordKey(pos[0]-1, pos[1], pos[2])] != nil

			count := 0
			for _, b := range []bool{above, below, north, south, east, west} {
				if b {
					count++
				}
			}

			// Slabs for count <= 4
			if count <= 4 {
				if !above && below {
					if count <= 3 {
						if !north && south {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("north", "top")}})
							continue
						}
						if !south && north {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("south", "top")}})
							continue
						}
						if !east && west {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("east", "top")}})
							continue
						}
						if !west && east {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("west", "top")}})
							continue
						}
					}
					changes = append(changes, blockChange{pos, &voxel{blockType: BlockSlabBot}})
					continue
				}
				if above && !below {
					if count <= 3 {
						if !north && south {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("north", "bottom")}})
							continue
						}
						if !south && north {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("south", "bottom")}})
							continue
						}
						if !east && west {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("east", "bottom")}})
							continue
						}
						if !west && east {
							changes = append(changes, blockChange{pos, &voxel{blockType: BlockStair, props: stairProps("west", "bottom")}})
							continue
						}
					}
					changes = append(changes, blockChange{pos, &voxel{blockType: BlockSlabTop}})
					continue
				}
			}
		}

		for _, c := range changes {
			grid[c.pos] = c.v
		}

		// Pass 2b: Backing for partial blocks
		// For every slab/stair, check all 6 neighbors; any that are empty AND
		// were inside the original ellipsoid get filled with a solid envelope block.
		// Run up to 3 iterations.
		for iter := 0; iter < 3; iter++ {
			var backing []coord
			for pos, v := range grid {
				if v.blockType != BlockSlabTop && v.blockType != BlockSlabBot && v.blockType != BlockStair {
					continue
				}
				for _, d := range dirs {
					nb := coordKey(pos[0]+d[0], pos[1]+d[1], pos[2]+d[2])
					if grid[nb] == nil && insideSet[nb] {
						backing = append(backing, nb)
					}
				}
			}
			if len(backing) == 0 {
				break
			}
			for _, nb := range backing {
				if grid[nb] == nil {
					grid[nb] = &voxel{blockType: BlockWool}
				}
			}
		}
	}

	// Pass 3: Ribbing
	if p.RibEnabled && p.RibSpacing > 0 {
		for pos, v := range grid {
			if v.blockType == BlockWool && pos[0]%p.RibSpacing == 0 {
				v.blockType = BlockLog
			}
		}
	}

	// Pass 4: Keel — per-X-column, find lowest-Y block at center Z and extend downward
	// (In balloon coords low Y = top, but renderer inverts Y, so low Y renders at bottom)
	if p.KeelEnabled && p.KeelDepth > 0 {
		midZ := int(cZ)
		for x := 0; x < sizeX; x++ {
			minY := -1
			for y := 0; y < sizeY; y++ {
				if grid[coordKey(x, y, midZ)] != nil {
					minY = y
					break
				}
			}
			if minY >= 0 {
				for dy := 1; dy <= p.KeelDepth; dy++ {
					grid[coordKey(x, minY-dy, midZ)] = &voxel{blockType: BlockLog}
				}
			}
		}
	}

	// Pass 5: Tail fins
	// Vertical fin on top (in renderer): find highest-Y block (renders at top) and extend upward
	if p.FinEnabled && p.FinHeight > 0 && p.FinLength > 0 {
		midZ := int(cZ)

		finStartX := sizeX - p.FinLength - 1

		for x := max(0, finStartX); x < sizeX; x++ {
			progress := float64(x-finStartX) / float64(p.FinLength)
			h := int(math.Ceil(float64(p.FinHeight) * (1 - progress)))

			botY := -1
			for y := sizeY - 1; y >= 0; y-- {
				if grid[coordKey(x, y, midZ)] != nil {
					botY = y
					break
				}
			}
			if botY >= 0 {
				for dy := 1; dy <= h; dy++ {
					grid[coordKey(x, botY+dy, midZ)] = &voxel{blockType: BlockPlank}
				}
			}
		}

		// Horizontal fins at center Y, extending +/- dz
		midY := int(cY)
		for x := max(0, sizeX-p.FinLength-1); x < sizeX; x++ {
			progress := float64(x-(sizeX-p.FinLength-1)) / float64(p.FinLength)
			w := int(math.Ceil(float64(p.FinHeight) * 0.7 * (1 - progress)))
			for dz := 1; dz <= w; dz++ {
				if grid[coordKey(x, midY, midZ)] != nil {
					grid[coordKey(x, midY, midZ-dz)] = &voxel{blockType: BlockPlank}
					grid[coordKey(x, midY, midZ+dz)] = &voxel{blockType: BlockPlank}
				}
			}
		}
	}

	// Pass 5b: Side fins — vertical fins on the sides of the tail
	if p.SideFinEnabled && p.FinHeight > 0 && p.FinLength > 0 {
		midY := int(cY)
		finStartX := sizeX - p.FinLength - 1

		for x := max(0, finStartX); x < sizeX; x++ {
			progress := float64(x-finStartX) / float64(p.FinLength)
			h := int(math.Ceil(float64(p.FinHeight) * 0.7 * (1 - progress)))

			// Find outermost Z on each side at center Y
			minZ, maxZ := -1, -1
			for z := 0; z < sizeZ; z++ {
				if grid[coordKey(x, midY, z)] != nil {
					if minZ < 0 {
						minZ = z
					}
					maxZ = z
				}
			}
			if minZ >= 0 {
				for dz := 1; dz <= h; dz++ {
					grid[coordKey(x, midY, minZ-dz)] = &voxel{blockType: BlockPlank}
					grid[coordKey(x, midY, maxZ+dz)] = &voxel{blockType: BlockPlank}
				}
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
		b := Block{X: pos[0], Y: ny, Z: pos[2], Type: v.blockType}
		if v.props != nil {
			b.Props = v.props
		}
		blocks = append(blocks, b)
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

	woodType := p.FrameWoodType
	if p.FrameMaterial == "andesite_casing" {
		woodType = "spruce"
	}

	return &GenerateResult{
		Blocks: blocks,
		SizeX:  actualMaxX + 1,
		SizeY:  actualMaxY + 1,
		SizeZ:  actualMaxZ + 1,
		Materials: MaterialConfig{
			WoodType:         woodType,
			EnvelopeMaterial: p.EnvelopeMaterial,
			EnvelopeColor:    p.EnvelopeColor,
			FrameMaterial:    p.FrameMaterial,
		},
	}, nil
}

func stairProps(facing, half string) map[string]string {
	return map[string]string{
		"facing":      facing,
		"half":        half,
		"shape":       "straight",
		"waterlogged": "false",
	}
}
