package generator

import (
	"fmt"
	"math"
)

type HullParams struct {
	Length         int     `json:"length"`
	Beam           int     `json:"beam"`
	Depth          int     `json:"depth"`
	BottomPinch    float64 `json:"bottomPinch"`
	HullFlare      float64 `json:"hullFlare"`
	FlareCurve     float64 `json:"flareCurve"`
	Tumblehome     float64 `json:"tumblehome"`
	TumbleCurve    float64 `json:"tumbleCurve"`
	SheerCurve     float64 `json:"sheerCurve"`
	SheerCurveExp  float64 `json:"sheerCurveExp"`
	BowLength      int     `json:"bowLength"`
	BowSharpness   float64 `json:"bowSharpness"`
	SternStyle     string  `json:"sternStyle"`
	SternLength    int     `json:"sternLength"`
	SternSharpness float64 `json:"sternSharpness"`
	HasRailings      bool    `json:"hasRailings"`
	HasTrim          bool    `json:"hasTrim"`
	CastleHeight     int     `json:"castleHeight"`
	CastleLength     int     `json:"castleLength"`
	ForecastleHeight int     `json:"forecastleHeight"`
	ForecastleLength int     `json:"forecastleLength"`
	HasGunPorts      bool    `json:"hasGunPorts"`
	GunPortSpacing   int     `json:"gunPortSpacing"`
}

func (p *HullParams) Validate() error {
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

	clampInt(&p.Length, 20, 200)
	clampInt(&p.Beam, 4, 40)
	clampInt(&p.Depth, 3, 20)
	clampFloat(&p.BottomPinch, 0.1, 0.7)
	clampFloat(&p.HullFlare, 0, 0.6)
	clampFloat(&p.FlareCurve, 1.2, 4.0)
	clampFloat(&p.Tumblehome, 0, 0.4)
	clampFloat(&p.TumbleCurve, 1.5, 5.0)
	clampFloat(&p.SheerCurve, 0, 0.75)
	clampFloat(&p.SheerCurveExp, 1.0, 4.0)
	clampInt(&p.BowLength, 2, 40)
	clampFloat(&p.BowSharpness, 0.4, 2.5)
	clampInt(&p.SternLength, 2, 30)
	clampFloat(&p.SternSharpness, 0.2, 2.0)

	clampInt(&p.CastleHeight, 0, 6)
	clampInt(&p.CastleLength, 0, 30)
	if p.CastleLength > p.Length*55/100 {
		p.CastleLength = p.Length * 55 / 100
	}
	clampInt(&p.ForecastleHeight, 0, 3)
	clampInt(&p.ForecastleLength, 0, 20)
	if p.ForecastleLength > p.Length*50/100 {
		p.ForecastleLength = p.Length * 50 / 100
	}
	clampInt(&p.GunPortSpacing, 2, 8)

	if p.SternStyle != "square" && p.SternStyle != "round" && p.SternStyle != "pointed" {
		p.SternStyle = "round"
	}
	if p.BowLength > p.Length/2 {
		p.BowLength = p.Length / 2
	}
	if p.SternLength > p.Length/2 {
		p.SternLength = p.Length / 2
	}
	if p.Length < 20 {
		return fmt.Errorf("length must be at least 20")
	}
	return nil
}

func GenerateHull(p HullParams) (*GenerateResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	halfBeam := float64(p.Beam) / 2
	depth := float64(p.Depth)
	length := float64(p.Length)

	bowStart := length - float64(p.BowLength)
	sternEnd := float64(p.SternLength)

	crossSectionFactor := func(yNorm float64) float64 {
		if yNorm < 0 {
			yNorm = 0
		}
		if yNorm > 1 {
			yNorm = 1
		}
		base := p.BottomPinch + (1-p.BottomPinch)*math.Pow(yNorm, 0.6)
		flare := p.HullFlare * math.Pow(yNorm, p.FlareCurve)
		tumble := p.Tumblehome * math.Pow(yNorm, p.TumbleCurve)
		result := base + flare - tumble
		if result < 0.12 {
			result = 0.12
		}
		return result
	}

	longitudinalFactor := func(z float64) float64 {
		if z < sternEnd {
			t := z / sternEnd
			if t < 0 {
				t = 0
			}
			switch p.SternStyle {
			case "square":
				f := math.Pow(t, p.SternSharpness)
				if f < 0.72 {
					f = 0.72
				}
				return f
			case "round":
				return math.Pow(t, p.SternSharpness*0.55)
			case "pointed":
				return math.Pow(t, p.SternSharpness)
			}
		}
		if z > bowStart {
			t := (z - bowStart) / float64(p.BowLength)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			return math.Pow(1-t, p.BowSharpness)
		}
		return 1
	}

	deckYAt := func(z float64) float64 {
		base := depth
		// Sheer curve
		mid := length / 2
		distFromMid := math.Abs(z - mid)
		normalizedDist := distFromMid / mid
		sheer := p.SheerCurve * math.Pow(normalizedDist, p.SheerCurveExp) * depth * 0.5
		deckY := base + sheer

		// Aft castle plateau
		if p.CastleHeight > 0 && p.CastleLength > 0 {
			castleStart := length - float64(p.CastleLength) - float64(p.SternLength)
			if z >= castleStart {
				deckY += float64(p.CastleHeight)
			}
		}

		// Forecastle plateau
		if p.ForecastleHeight > 0 && p.ForecastleLength > 0 {
			foreEnd := float64(p.BowLength) + float64(p.ForecastleLength)
			if z < foreEnd && z >= float64(p.BowLength) {
				deckY += float64(p.ForecastleHeight)
			}
		}

		return deckY
	}

	halfWidthAt := func(y int, z int) float64 {
		yNorm := float64(y) / depth
		return crossSectionFactor(yNorm) * longitudinalFactor(float64(z)) * halfBeam
	}

	sizeX := p.Beam + 2
	castleExtra := p.CastleHeight
	if p.ForecastleHeight > castleExtra {
		castleExtra = p.ForecastleHeight
	}
	sizeY := p.Depth + int(math.Ceil(p.SheerCurve*depth*0.5)) + castleExtra + 2
	sizeZ := p.Length + 1

	centerX := sizeX / 2

	type voxel struct {
		blockType int
		props     map[string]string
	}
	grid := make(map[[3]int]*voxel)

	set := func(x, y, z, bt int) {
		if x >= 0 && x < sizeX && y >= 0 && y < sizeY && z >= 0 && z < sizeZ {
			grid[[3]int{x, y, z}] = &voxel{blockType: bt}
		}
	}

	get := func(x, y, z int) *voxel {
		return grid[[3]int{x, y, z}]
	}

	// Pass 1: fill hull interior
	for z := 0; z < sizeZ; z++ {
		deckH := deckYAt(float64(z))
		for y := 0; y < sizeY; y++ {
			if float64(y) > deckH {
				continue
			}
			hw := halfWidthAt(y, z)
			for x := 0; x < sizeX; x++ {
				dx := math.Abs(float64(x) - float64(centerX))
				if dx <= hw {
					set(x, y, z, BlockPlank)
				}
			}
		}
	}

	// Pass 2: shell — keep only outer surface + deck
	dirs := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	surface := make(map[[3]int]bool)
	for pos := range grid {
		for _, d := range dirs {
			nb := [3]int{pos[0] + d[0], pos[1] + d[1], pos[2] + d[2]}
			if get(nb[0], nb[1], nb[2]) == nil {
				surface[pos] = true
				break
			}
		}
	}
	for pos := range grid {
		if !surface[pos] {
			delete(grid, pos)
		}
	}

	// Also add full deck (top surface)
	for z := 0; z < sizeZ; z++ {
		deckH := int(math.Round(deckYAt(float64(z))))
		for x := 0; x < sizeX; x++ {
			hw := halfWidthAt(deckH, z)
			dx := math.Abs(float64(x) - float64(centerX))
			if dx <= hw {
				set(x, deckH, z, BlockPlank)
			}
		}
	}

	// Pass 8: gunwale trim (slabs at deck edge)
	if p.HasTrim {
		for z := 0; z < sizeZ; z++ {
			deckH := int(math.Round(deckYAt(float64(z))))
			hw := halfWidthAt(deckH, z)
			if hw < 1 {
				continue
			}
			leftX := centerX - int(math.Round(hw))
			rightX := centerX + int(math.Round(hw))
			if get(leftX, deckH, z) != nil {
				set(leftX, deckH+1, z, BlockSlabBot)
			}
			if get(rightX, deckH, z) != nil {
				set(rightX, deckH+1, z, BlockSlabBot)
			}
		}
	}

	// Pass 9: railings
	if p.HasRailings {
		for z := 0; z < sizeZ; z++ {
			deckH := int(math.Round(deckYAt(float64(z))))
			hw := halfWidthAt(deckH, z)
			if hw < 1 {
				continue
			}
			railY := deckH + 1
			if p.HasTrim {
				railY = deckH + 2
			}
			leftX := centerX - int(math.Round(hw))
			rightX := centerX + int(math.Round(hw))
			if get(leftX, deckH, z) != nil || get(leftX, deckH+1, z) != nil {
				set(leftX, railY, z, BlockFence)
			}
			if get(rightX, deckH, z) != nil || get(rightX, deckH+1, z) != nil {
				set(rightX, railY, z, BlockFence)
			}
		}
	}

	// Pass 10: gun ports
	if p.HasGunPorts && p.GunPortSpacing > 0 {
		for z := 0; z < sizeZ; z++ {
			if z%p.GunPortSpacing != 0 {
				continue
			}
			deckH := int(math.Round(deckYAt(float64(z))))
			gunY := deckH - 1
			if gunY < 1 {
				continue
			}
			hw := halfWidthAt(gunY, z)
			if hw < 1 {
				continue
			}
			leftX := centerX - int(math.Round(hw))
			rightX := centerX + int(math.Round(hw))
			if get(leftX, gunY, z) != nil {
				set(leftX, gunY, z, BlockTrapdoor)
			}
			if get(rightX, gunY, z) != nil {
				set(rightX, gunY, z, BlockTrapdoor)
			}
		}
	}

	// Build result
	var blocks []Block
	actualMaxX, actualMaxY, actualMaxZ := 0, 0, 0
	for pos, v := range grid {
		blocks = append(blocks, Block{X: pos[0], Y: pos[1], Z: pos[2], Type: v.blockType})
		if pos[0] > actualMaxX {
			actualMaxX = pos[0]
		}
		if pos[1] > actualMaxY {
			actualMaxY = pos[1]
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
