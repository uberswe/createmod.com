package generator

import (
	"fmt"
	"math"
)

type PropellerParams struct {
	Version       int     `json:"version"`
	Blades        int     `json:"blades"`
	Length        int     `json:"length"`
	RootChord     int     `json:"rootChord"`
	TipChord      int     `json:"tipChord"`
	SweepDegrees  float64 `json:"sweepDegrees"`
	Swept         bool    `json:"swept"`
	AirfoilShape  string  `json:"airfoilShape"`
	BladeMaterial string  `json:"bladeMaterial"`
	BladeColor    string  `json:"bladeColor"`
	Rotation      float64 `json:"rotation"`
	Orientation   string  `json:"orientation"`
}

func (p *PropellerParams) Validate() error {
	if p.Version == 0 {
		p.Version = CurrentVersion
	}
	if p.Blades < 2 || p.Blades > 12 {
		return fmt.Errorf("blades must be between 2 and 12")
	}
	if p.Length < 3 || p.Length > 50 {
		return fmt.Errorf("length must be between 3 and 50")
	}
	if p.RootChord < 1 || p.RootChord > 40 {
		return fmt.Errorf("rootChord must be between 1 and 40")
	}
	if p.TipChord < 0 || p.TipChord > 40 {
		return fmt.Errorf("tipChord must be between 0 and 40")
	}
	if p.SweepDegrees < 0 || p.SweepDegrees > 90 {
		return fmt.Errorf("sweepDegrees must be between 0 and 90")
	}
	if p.AirfoilShape != "linear" && p.AirfoilShape != "curved" {
		p.AirfoilShape = "linear"
	}
	if p.BladeMaterial != "wool" && p.BladeMaterial != "sail" {
		p.BladeMaterial = "wool"
	}
	if !isValidWoolColor(p.BladeColor) {
		p.BladeColor = "white"
	}
	if p.Rotation < 0 || p.Rotation > 360 {
		p.Rotation = 0
	}
	if p.Orientation != "horizontal" && p.Orientation != "vertical" {
		p.Orientation = "horizontal"
	}
	return nil
}

func symRound(v float64) int {
	if v >= 0 {
		return int(math.Floor(v + 0.5))
	}
	return -int(math.Floor(-v + 0.5))
}

func sampleRange(lo, hi, step float64) []float64 {
	var out []float64
	for v := lo; v <= hi+step*0.01; v += step {
		out = append(out, v)
	}
	return out
}

func GeneratePropeller(p PropellerParams) (*GenerateResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	bladeType := BlockWool
	if p.BladeMaterial == "sail" {
		bladeType = BlockSail
	}

	seen := make(map[[3]int]bool)
	var blocks []Block

	addBlock := func(x, y, z int) {
		key := [3]int{x, y, z}
		if !seen[key] {
			seen[key] = true
			blocks = append(blocks, Block{X: x, Y: y, Z: z, Type: bladeType})
		}
	}

	sweepRad := p.SweepDegrees * math.Pi / 180
	rotationRad := p.Rotation * math.Pi / 180

	for b := 0; b < p.Blades; b++ {
		angle := float64(b)/float64(p.Blades)*2*math.Pi + rotationRad

		for _, r := range sampleRange(0, float64(p.Length), 0.35) {
			t := r / float64(p.Length)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}

			chord := float64(p.RootChord) + (float64(p.TipChord)-float64(p.RootChord))*t
			if p.AirfoilShape == "curved" {
				chord += math.Sin(t*math.Pi) * math.Min(1.3, float64(p.RootChord)*0.4)
			}

			if chord < 0.5 {
				continue
			}

			localAngle := angle
			if p.Swept {
				localAngle += sweepRad * t
			}

			halfC := math.Max(0, (chord-1)/2)
			cosA := math.Cos(localAngle)
			sinA := math.Sin(localAngle)

			for _, w := range sampleRange(-halfC, halfC, 0.35) {
				bx := symRound(r*cosA - w*sinA)
				bz := symRound(r*sinA + w*cosA)
				addBlock(bx, 0, bz)
			}
		}
	}

	// Vertical orientation: rotate the XZ disc into the XY plane
	if p.Orientation == "vertical" {
		for i := range blocks {
			blocks[i].Y = blocks[i].Z
			blocks[i].Z = 0
		}
	}

	// Normalize to positive coordinates
	minX, minY, minZ := 0, 0, 0
	maxX, maxY, maxZ := 0, 0, 0
	for _, b := range blocks {
		if b.X < minX {
			minX = b.X
		}
		if b.Y < minY {
			minY = b.Y
		}
		if b.Z < minZ {
			minZ = b.Z
		}
		if b.X > maxX {
			maxX = b.X
		}
		if b.Y > maxY {
			maxY = b.Y
		}
		if b.Z > maxZ {
			maxZ = b.Z
		}
	}

	for i := range blocks {
		blocks[i].X -= minX
		blocks[i].Y -= minY
		blocks[i].Z -= minZ
	}

	sailFacing := "up"
	if p.Orientation == "vertical" {
		sailFacing = "north"
	}

	return &GenerateResult{
		Blocks: blocks,
		SizeX:  maxX - minX + 1,
		SizeY:  maxY - minY + 1,
		SizeZ:  maxZ - minZ + 1,
		Materials: MaterialConfig{
			BladeMaterial: p.BladeMaterial,
			BladeColor:    p.BladeColor,
			SailFacing:    sailFacing,
			Orientation:   p.Orientation,
		},
	}, nil
}
