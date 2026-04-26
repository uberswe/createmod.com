package generator

import (
	"fmt"
	"math"
)

type PropellerParams struct {
	Blades       int     `json:"blades"`
	Length       int     `json:"length"`
	RootChord    int     `json:"rootChord"`
	TipChord     int     `json:"tipChord"`
	SweepDegrees float64 `json:"sweepDegrees"`
	Swept        bool    `json:"swept"`
	AirfoilShape string  `json:"airfoilShape"`
}

func (p *PropellerParams) Validate() error {
	if p.Blades < 2 || p.Blades > 12 {
		return fmt.Errorf("blades must be between 2 and 12")
	}
	if p.Length < 3 || p.Length > 200 {
		return fmt.Errorf("length must be between 3 and 200")
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
	return nil
}

func hubRadius(length int) int {
	r := int(math.Ceil(float64(length) / 30))
	if r < 2 {
		return 2
	}
	return r
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

	seen := make(map[[3]int]bool)
	var blocks []Block

	addBlock := func(x, y, z int) {
		key := [3]int{x, y, z}
		if !seen[key] {
			seen[key] = true
			blocks = append(blocks, Block{X: x, Y: y, Z: z, Type: BlockWool})
		}
	}

	hr := hubRadius(p.Length)
	bladeStart := float64(hr)

	// Hub
	if p.RootChord >= 2 {
		for ix := -(hr - 1); ix <= hr-1; ix++ {
			for iz := -(hr - 1); iz <= hr-1; iz++ {
				addBlock(ix, 0, iz)
			}
		}
	} else {
		for ix := -(hr - 1); ix <= hr-1; ix++ {
			for iz := -(hr - 1); iz <= hr-1; iz++ {
				if float64(ix*ix+iz*iz) <= float64((hr-1)*(hr-1))+0.25 {
					addBlock(ix, 0, iz)
				}
			}
		}
	}

	sweepRad := p.SweepDegrees * math.Pi / 180

	for b := 0; b < p.Blades; b++ {
		angle := float64(b) / float64(p.Blades) * 2 * math.Pi

		for _, r := range sampleRange(bladeStart, float64(p.Length), 0.35) {
			t := (r - bladeStart) / (float64(p.Length) - bladeStart)
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}

			chord := float64(p.RootChord) + (float64(p.TipChord)-float64(p.RootChord))*t
			if p.AirfoilShape == "curved" {
				chord += math.Sin(t*math.Pi) * 1.3
			}

			localAngle := angle
			if p.Swept {
				localAngle += sweepRad * t
			}

			halfC := chord / 2
			cosA := math.Cos(localAngle)
			sinA := math.Sin(localAngle)

			for _, w := range sampleRange(-halfC, halfC, 0.35) {
				bx := symRound(r*cosA - w*sinA)
				bz := symRound(r*sinA + w*cosA)
				addBlock(bx, 0, bz)
			}
		}
	}

	// Normalize to positive coordinates
	minX, minZ := 0, 0
	maxX, maxZ := 0, 0
	for _, b := range blocks {
		if b.X < minX {
			minX = b.X
		}
		if b.Z < minZ {
			minZ = b.Z
		}
		if b.X > maxX {
			maxX = b.X
		}
		if b.Z > maxZ {
			maxZ = b.Z
		}
	}

	for i := range blocks {
		blocks[i].X -= minX
		blocks[i].Z -= minZ
	}

	return &GenerateResult{
		Blocks: blocks,
		SizeX:  maxX - minX + 1,
		SizeY:  1,
		SizeZ:  maxZ - minZ + 1,
	}, nil
}
