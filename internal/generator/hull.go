package generator

import (
	"fmt"
	"math"
	"sort"
)

type HullParams struct {
	Version        int     `json:"version"`
	WoodType       string  `json:"woodType"`
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
	BowKeelRise    float64 `json:"bowKeelRise"`
	BowKeelLength  int     `json:"bowKeelLength"`
	BowCurve       float64 `json:"bowCurve"`
	SternStyle     string  `json:"sternStyle"`
	SternLength    int     `json:"sternLength"`
	SternSharpness float64 `json:"sternSharpness"`
	SternKeelRise  float64 `json:"sternKeelRise"`
	SternKeelLength int    `json:"sternKeelLength"`
	SternOverhang  float64 `json:"sternOverhang"`
	KeelCurve      float64 `json:"keelCurve"`
	CastleBlend    int     `json:"castleBlend"`
	HasRailings      bool    `json:"hasRailings"`
	HasTrim          bool    `json:"hasTrim"`
	HasWindows       bool    `json:"hasWindows"`
	CastleHeight     int     `json:"castleHeight"`
	CastleLength     int     `json:"castleLength"`
	ForecastleHeight int     `json:"forecastleHeight"`
	ForecastleLength int     `json:"forecastleLength"`
	HasGunPorts      bool    `json:"hasGunPorts"`
	GunPortRow       int     `json:"gunPortRow"`
	GunPortSpacing   int     `json:"gunPortSpacing"`
	MidWidthBias     float64 `json:"midWidthBias"`
}

func (p *HullParams) Validate() error {
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
	clampFloat(&p.BowKeelRise, 0, 1.5)
	clampInt(&p.BowKeelLength, 0, 40)
	clampFloat(&p.BowCurve, -1.0, 1.0)
	clampInt(&p.SternLength, 2, 30)
	clampFloat(&p.SternSharpness, 0.2, 2.0)
	clampFloat(&p.SternKeelRise, 0, 1.5)
	clampInt(&p.SternKeelLength, 0, 30)
	clampFloat(&p.SternOverhang, 0, 1.0)
	clampFloat(&p.KeelCurve, 0.7, 3.5)
	clampInt(&p.CastleBlend, 2, 12)

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
	clampInt(&p.GunPortRow, 1, 6)
	clampInt(&p.GunPortSpacing, 2, 8)
	clampFloat(&p.MidWidthBias, 0, 1.0)

	if !isValidWoodType(p.WoodType) {
		p.WoodType = "spruce"
	}
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

	// Apply defaults for zero-value fields that need non-zero defaults
	if p.KeelCurve == 0 {
		p.KeelCurve = 1.7
	}
	if p.CastleBlend == 0 {
		p.CastleBlend = 4
	}
	if p.GunPortRow == 0 {
		p.GunPortRow = 2
	}

	return nil
}

func smoothstep(t float64) float64 {
	x := t
	if x < 0 {
		x = 0
	}
	if x > 1 {
		x = 1
	}
	return x * x * (3 - 2*x)
}

func GenerateHull(p HullParams) (*GenerateResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	L := p.Length
	D := p.Depth
	depth := float64(D)
	length := float64(L)

	// Cross-section width factor at normalized Y. yNorm can exceed 1 for castle
	// sections above main deck; the function extrapolates with a steep taper.
	crossSectionFactor := func(yNorm float64) float64 {
		yc := yNorm
		if yc > 1 {
			yc = 1
		}
		if yc < 0 {
			yc = 0
		}
		base := p.BottomPinch + (1-p.BottomPinch)*math.Pow(yc, 0.6)
		flare := p.HullFlare * math.Pow(yc, p.FlareCurve)
		tumble := p.Tumblehome * math.Pow(yc, p.TumbleCurve)
		// Above-deck taper: castle sections taper inward with height
		above := yNorm - 1
		if above < 0 {
			above = 0
		}
		castleTaper := above*0.32 + above*above*0.18
		result := base + flare - tumble - castleTaper
		if result < 0.12 {
			result = 0.12
		}
		return result
	}

	longitudinalFactor := func(zNorm float64) float64 {
		bowStart := 1.0 - float64(p.BowLength)/length
		sternEnd := float64(p.SternLength) / length

		if zNorm <= sternEnd {
			t := zNorm / math.Max(sternEnd, 0.001)
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
				return math.Pow(math.Max(t, 0), p.SternSharpness*0.55)
			default: // pointed
				return math.Pow(t, p.SternSharpness)
			}
		}
		if zNorm >= bowStart {
			t := (1 - zNorm) / math.Max(1-bowStart, 0.001)
			base := math.Pow(math.Max(t, 0), p.BowSharpness)
			// BowCurve: negative = concave (clipper), positive = convex (bluff)
			if p.BowCurve != 0 {
				if p.BowCurve > 0 {
					convex := math.Sqrt(math.Max(t, 0))
					base = base*(1-p.BowCurve) + convex*p.BowCurve
				} else {
					concave := t * t * t
					base = base*(1+p.BowCurve) + concave*(-p.BowCurve)
				}
			}
			return base
		}
		// MidWidthBias: shift widest point aft (0=centered, 1=fully aft)
		if p.MidWidthBias > 0 {
			midNorm := sternEnd + (bowStart-sternEnd)*(0.5-p.MidWidthBias*0.35)
			if zNorm < midNorm {
				t := (zNorm - sternEnd) / math.Max(midNorm-sternEnd, 0.001)
				return 0.85 + 0.15*math.Pow(t, 0.6)
			}
		}
		return 1
	}

	halfWidthAt := func(y, z int) float64 {
		yNorm := float64(y) / math.Max(depth, 1) // no clamp -- allow castle extrapolation
		zNorm := float64(z) / math.Max(length-1, 1)
		base := crossSectionFactor(yNorm) * longitudinalFactor(zNorm) * (float64(p.Beam) / 2)
		// SternOverhang: widen above-deck stern region outward
		if p.SternOverhang > 0 && yNorm > 1.0 && zNorm < float64(p.SternLength)/length {
			overhangBoost := p.SternOverhang * 0.3 * (yNorm - 1.0)
			base += overhangBoost * (float64(p.Beam) / 2)
		}
		return base
	}

	// Keel rise at a given Z position
	keelYAt := func(z int) int {
		zNorm := float64(z) / math.Max(length-1, 1)
		curve := p.KeelCurve
		rise := 0.0

		if p.BowKeelRise > 0 && p.BowKeelLength > 0 {
			start := 1.0 - float64(p.BowKeelLength)/length
			if zNorm > start {
				t := (zNorm - start) / math.Max(1-start, 0.001)
				r := math.Pow(t, curve) * p.BowKeelRise
				if r > rise {
					rise = r
				}
			}
		}
		if p.SternKeelRise > 0 && p.SternKeelLength > 0 {
			end := float64(p.SternKeelLength) / length
			if zNorm < end {
				t := (end - zNorm) / math.Max(end, 0.001)
				r := math.Pow(t, curve) * p.SternKeelRise
				if r > rise {
					rise = r
				}
			}
		}
		return int(math.Round(rise * depth))
	}

	// Deck Y at a given Z, including sheer curve + castle/forecastle
	deckYAt := func(z int) int {
		y := depth

		// Sheer curve
		if p.SheerCurve > 0 {
			zNorm := float64(z) / math.Max(length-1, 1)
			t := math.Abs(zNorm-0.5) * 2 // 0 at mid, 1 at ends
			y += p.SheerCurve * depth * math.Pow(t, p.SheerCurveExp)
		}

		// Aftcastle plateau (stern region)
		if p.CastleHeight > 0 && p.CastleLength > 0 {
			cL := p.CastleLength
			maxCL := int(math.Floor(length * 0.55))
			if cL > maxCL {
				cL = maxCL
			}
			blend := int(math.Floor(float64(cL) * 0.55))
			if blend > p.CastleBlend {
				blend = p.CastleBlend
			}
			if blend < 2 {
				blend = 2
			}
			if z < cL-blend {
				y += float64(p.CastleHeight)
			} else if z < cL {
				t := float64(z-(cL-blend)) / float64(blend)
				y += float64(p.CastleHeight) * (1 - smoothstep(t))
			}
		}

		// Forecastle plateau (bow region)
		if p.ForecastleHeight > 0 && p.ForecastleLength > 0 {
			fL := p.ForecastleLength
			maxFL := int(math.Floor(length * 0.5))
			if fL > maxFL {
				fL = maxFL
			}
			blend := int(math.Floor(float64(fL) * 0.55))
			if blend > p.CastleBlend {
				blend = p.CastleBlend
			}
			if blend < 2 {
				blend = 2
			}
			zFromBow := L - 1 - z
			if zFromBow < fL-blend {
				y += float64(p.ForecastleHeight)
			} else if zFromBow < fL {
				t := float64(zFromBow-(fL-blend)) / float64(blend)
				y += float64(p.ForecastleHeight) * (1 - smoothstep(t))
			}
		}

		return int(math.Round(y))
	}

	// --- Pass 1: build hull volume mask
	type coord [3]int
	key := func(x, y, z int) coord { return coord{x, y, z} }

	inHull := make(map[coord]bool)
	keelYArr := make([]int, L)
	deckYArr := make([]int, L)
	maxDeckY := D

	for z := 0; z < L; z++ {
		deckYArr[z] = deckYAt(z)
		if deckYArr[z] > maxDeckY {
			maxDeckY = deckYArr[z]
		}
	}

	// hwArr[y][z] = half-width int
	hwArr := make([][]int, maxDeckY+1)
	for y := 0; y <= maxDeckY; y++ {
		hwArr[y] = make([]int, L)
		for z := 0; z < L; z++ {
			hwArr[y][z] = -1
		}
	}

	for z := 0; z < L; z++ {
		keelYArr[z] = keelYAt(z)
		topY := deckYArr[z]
		for y := keelYArr[z]; y <= topY; y++ {
			hw := halfWidthAt(y, z)
			if hw < 0.15 {
				continue
			}
			maxX := int(math.Max(0, math.Round(hw-0.0001)))
			if y <= maxDeckY {
				hwArr[y][z] = maxX
			}
			for x := -maxX; x <= maxX; x++ {
				inHull[key(x, y, z)] = true
			}
		}
	}

	has := func(x, y, z int) bool {
		return inHull[key(x, y, z)]
	}

	// --- Pass 2: shell (planks only on exterior surface + solid deck)
	type blockEntry struct {
		x, y, z int
		name    string
		props   map[string]string
	}
	blocks := make(map[coord]*blockEntry)

	set := func(x, y, z int, name string, props map[string]string) {
		blocks[key(x, y, z)] = &blockEntry{x: x, y: y, z: z, name: name, props: props}
	}
	get := func(x, y, z int) *blockEntry {
		return blocks[key(x, y, z)]
	}

	for k := range inHull {
		x, y, z := k[0], k[1], k[2]
		exposed := !has(x-1, y, z) || !has(x+1, y, z) ||
			!has(x, y-1, z) || !has(x, y+1, z) ||
			!has(x, y, z-1) || !has(x, y, z+1)
		isDeck := y == deckYArr[z]
		if exposed || isDeck {
			set(x, y, z, "minecraft:spruce_planks", nil)
		}
	}

	// --- Pass 3: lateral flare stairs (hull widens going up)
	for z := 0; z < L; z++ {
		for y := keelYArr[z]; y < deckYArr[z]; y++ {
			hwHere := -1
			if y <= maxDeckY && y >= 0 {
				hwHere = hwArr[y][z]
			}
			hwUp := -1
			if y+1 <= maxDeckY && y+1 >= 0 {
				hwUp = hwArr[y+1][z]
			}
			if hwUp <= hwHere {
				continue
			}
			for xNew := hwHere + 1; xNew <= hwUp; xNew++ {
				if has(xNew, y, z) {
					continue
				}
				if !has(xNew, y+1, z) {
					continue
				}
				if existing := get(xNew, y, z); existing != nil && existing.name == "minecraft:spruce_planks" {
					continue
				}
				set(xNew, y, z, "minecraft:spruce_stairs", map[string]string{
					"facing": "east", "half": "top", "shape": "straight", "waterlogged": "false",
				})
				set(-xNew, y, z, "minecraft:spruce_stairs", map[string]string{
					"facing": "west", "half": "top", "shape": "straight", "waterlogged": "false",
				})
			}
		}
	}

	// --- Pass 4: longitudinal taper stairs (hull narrows along length)
	placeLongStair := func(x, y, z int, facing string) {
		if has(x, y, z) {
			return
		}
		existing := get(x, y, z)
		if existing != nil && existing.name == "minecraft:spruce_planks" {
			return
		}
		if existing != nil && existing.name == "minecraft:spruce_stairs" {
			return
		}
		set(x, y, z, "minecraft:spruce_stairs", map[string]string{
			"facing": facing, "half": "top", "shape": "straight", "waterlogged": "false",
		})
	}

	for y := 0; y <= maxDeckY; y++ {
		for z := 0; z < L; z++ {
			hwThis := -1
			if y <= maxDeckY && y >= 0 {
				hwThis = hwArr[y][z]
			}
			if hwThis < 0 {
				continue
			}
			// Forward (toward bow / +Z)
			hwForward := -1
			if z+1 < L {
				hwForward = hwArr[y][z+1]
			}
			if hwForward >= 0 && hwForward < hwThis {
				for x := hwForward + 1; x <= hwThis; x++ {
					placeLongStair(x, y, z+1, "south")
					if x != 0 {
						placeLongStair(-x, y, z+1, "south")
					}
				}
			}
			// Backward (toward stern / -Z)
			hwBack := -1
			if z > 0 {
				hwBack = hwArr[y][z-1]
			}
			if hwBack >= 0 && hwBack < hwThis && z > 0 {
				for x := hwBack + 1; x <= hwThis; x++ {
					placeLongStair(x, y, z-1, "north")
					if x != 0 {
						placeLongStair(-x, y, z-1, "north")
					}
				}
			}
		}
	}

	// --- Pass 5: keel-rise stairs
	for z := 0; z < L-1; z++ {
		k0 := keelYArr[z]
		k1 := keelYArr[z+1]
		if k1 == k0 {
			continue
		}
		var dir string
		var yFill, zFill, refZ int
		if k1 > k0 {
			dir = "bow"
			yFill = k1 - 1
			zFill = z + 1
			refZ = z + 1
		} else {
			dir = "stern"
			yFill = k0 - 1
			zFill = z
			refZ = z
		}
		hw := -1
		if yFill+1 <= maxDeckY && yFill+1 >= 0 && refZ >= 0 && refZ < L {
			hw = hwArr[yFill+1][refZ]
		}
		if hw < 0 {
			continue
		}
		for x := -hw; x <= hw; x++ {
			if has(x, yFill, zFill) {
				continue
			}
			if get(x, yFill, zFill) != nil {
				continue
			}
			facing := "south"
			if dir == "stern" {
				facing = "north"
			}
			set(x, yFill, zFill, "minecraft:spruce_stairs", map[string]string{
				"facing": facing, "half": "top", "shape": "straight", "waterlogged": "false",
			})
		}
	}

	// --- Pass 5.5: no stair stacking -- replace stair above stair with plank
	{
		type stairEntry struct {
			x, y, z int
		}
		var stairList []stairEntry
		for k, b := range blocks {
			if b.name == "minecraft:spruce_stairs" {
				stairList = append(stairList, stairEntry{k[0], k[1], k[2]})
			}
		}
		// Sort top-down
		sort.Slice(stairList, func(i, j int) bool {
			return stairList[i].y > stairList[j].y
		})
		for _, s := range stairList {
			below := get(s.x, s.y-1, s.z)
			if below != nil && below.name == "minecraft:spruce_stairs" {
				set(s.x, s.y, s.z, "minecraft:spruce_planks", nil)
			}
		}
	}

	// --- Pass 6: stern windows
	if p.HasWindows && p.CastleHeight >= 2 && p.CastleLength > 0 {
		z := 0
		wy := D + 1 // main-deck level + 1
		if deckYArr[z] > D {
			hwBack := -1
			if wy <= maxDeckY && wy >= 0 {
				hwBack = hwArr[wy][z]
			}
			if hwBack >= 1 {
				for x := -hwBack + 1; x <= hwBack-1; x += 2 {
					if existing := get(x, wy, z); existing != nil && existing.name == "minecraft:spruce_planks" {
						set(x, wy, z, "minecraft:spruce_trapdoor", map[string]string{
							"facing": "north", "half": "bottom", "open": "true",
							"powered": "false", "waterlogged": "false",
						})
					}
				}
			}
		}
	}

	// --- Pass 8/9 combined: gunwale trim (slabs) + fence railings
	{
		for z := 0; z < L; z++ {
			deckY := deckYArr[z]
			hw := -1
			if deckY <= maxDeckY && deckY >= 0 {
				hw = hwArr[deckY][z]
			}
			if hw < 1 {
				continue
			}
			y := deckY + 1

			wantTrim := p.HasTrim
			wantRail := p.HasRailings
			canInset := hw >= 2

			if wantTrim && wantRail && canInset {
				if get(hw, y, z) == nil && !has(hw, y, z) {
					set(hw, y, z, "minecraft:spruce_slab", map[string]string{
						"type": "bottom", "waterlogged": "false",
					})
				}
				if get(-hw, y, z) == nil && !has(-hw, y, z) {
					set(-hw, y, z, "minecraft:spruce_slab", map[string]string{
						"type": "bottom", "waterlogged": "false",
					})
				}
				set(hw-1, y, z, "minecraft:spruce_fence", map[string]string{
					"north": "false", "south": "false", "east": "false", "west": "false", "waterlogged": "false",
				})
				if hw-1 > 0 {
					set(-(hw-1), y, z, "minecraft:spruce_fence", map[string]string{
						"north": "false", "south": "false", "east": "false", "west": "false", "waterlogged": "false",
					})
				}
			} else if wantTrim && !wantRail {
				if get(hw, y, z) == nil && !has(hw, y, z) {
					set(hw, y, z, "minecraft:spruce_slab", map[string]string{
						"type": "bottom", "waterlogged": "false",
					})
				}
				if hw > 0 && get(-hw, y, z) == nil && !has(-hw, y, z) {
					set(-hw, y, z, "minecraft:spruce_slab", map[string]string{
						"type": "bottom", "waterlogged": "false",
					})
				}
			} else if wantRail {
				set(hw, y, z, "minecraft:spruce_fence", map[string]string{
					"north": "false", "south": "false", "east": "false", "west": "false", "waterlogged": "false",
				})
				if hw > 0 {
					set(-hw, y, z, "minecraft:spruce_fence", map[string]string{
						"north": "false", "south": "false", "east": "false", "west": "false", "waterlogged": "false",
					})
				}
			}
		}
	}

	// --- Defensive pass: never allow a fence directly over a slab
	for _, b := range blocks {
		if b.name != "minecraft:spruce_fence" {
			continue
		}
		below := get(b.x, b.y-1, b.z)
		if below != nil && below.name == "minecraft:spruce_slab" {
			delete(blocks, key(b.x, b.y-1, b.z))
		}
	}

	// --- Pass 10: gun ports
	if p.HasGunPorts && p.GunPortRow > 0 {
		midKeelY := keelYAt(L / 2)
		yPort := D - p.GunPortRow
		if midKeelY+1 > yPort {
			yPort = midKeelY + 1
		}
		for z := 3; z < L-3; z += p.GunPortSpacing {
			hw := -1
			if yPort >= 0 && yPort <= maxDeckY {
				hw = hwArr[yPort][z]
			}
			if hw < 1 {
				continue
			}
			set(hw, yPort, z, "minecraft:spruce_trapdoor", map[string]string{
				"facing": "east", "half": "bottom", "open": "true",
				"powered": "false", "waterlogged": "false",
			})
			if hw > 0 {
				set(-hw, yPort, z, "minecraft:spruce_trapdoor", map[string]string{
					"facing": "west", "half": "bottom", "open": "true",
					"powered": "false", "waterlogged": "false",
				})
			}
		}
	}

	// --- Pass 10.5: fence bridging for sheer/castle/taper transitions
	{
		isSupport := func(b *blockEntry) bool {
			if b == nil {
				return false
			}
			return b.name == "minecraft:spruce_planks" ||
				b.name == "minecraft:spruce_fence" ||
				b.name == "minecraft:spruce_stairs" ||
				b.name == "minecraft:spruce_slab"
		}
		addBridge := func(x, y, z int) bool {
			if blocks[key(x, y, z)] != nil {
				return false
			}
			if !isSupport(get(x, y-1, z)) {
				return false
			}
			set(x, y, z, "minecraft:spruce_fence", map[string]string{
				"north": "false", "south": "false", "east": "false", "west": "false", "waterlogged": "false",
			})
			return true
		}

		// Collect current fences
		var fences []*blockEntry
		for _, b := range blocks {
			if b.name == "minecraft:spruce_fence" {
				fences = append(fences, b)
			}
		}

		for _, f := range fences {
			for _, dz := range []int{-1, 1} {
				direct := get(f.x, f.y, f.z+dz)
				if direct != nil && direct.name == "minecraft:spruce_fence" {
					continue
				}
				for _, dy := range []int{-2, -1, 0, 1, 2} {
					for _, dx := range []int{-1, 0, 1} {
						if dx == 0 && dy == 0 {
							continue
						}
						n := get(f.x+dx, f.y+dy, f.z+dz)
						if n == nil || n.name != "minecraft:spruce_fence" {
							continue
						}
						commonY := f.y
						if n.y > commonY {
							commonY = n.y
						}
						if dx != 0 {
							addBridge(f.x+dx, commonY, f.z)
						}
						if dy != 0 {
							for yStack := n.y + 1; yStack <= commonY; yStack++ {
								addBridge(n.x, yStack, n.z)
							}
						}
					}
				}
			}
		}
	}

	// --- Pass 11: compute fence connection states
	{
		checkDir := func(bx, by, bz, dx, dz int) string {
			n := get(bx+dx, by, bz+dz)
			if n == nil {
				return "false"
			}
			if n.name == "minecraft:spruce_fence" ||
				n.name == "minecraft:spruce_planks" ||
				n.name == "minecraft:spruce_stairs" {
				return "true"
			}
			return "false"
		}

		for _, b := range blocks {
			if b.name != "minecraft:spruce_fence" {
				continue
			}
			props := b.props
			if props == nil {
				props = map[string]string{}
			}
			b.props = map[string]string{
				"east":        checkDir(b.x, b.y, b.z, 1, 0),
				"west":        checkDir(b.x, b.y, b.z, -1, 0),
				"south":       checkDir(b.x, b.y, b.z, 0, 1),
				"north":       checkDir(b.x, b.y, b.z, 0, -1),
				"waterlogged": "false",
			}
		}
	}

	// --- Pass 12: compute stair shapes (inner/outer corners)
	{
		type vec2 struct{ x, z int }
		facingVec := map[string]vec2{
			"south": {0, 1}, "north": {0, -1},
			"east": {1, 0}, "west": {-1, 0},
		}
		facingLeftOf := map[string]string{
			"south": "east", "north": "west", "east": "north", "west": "south",
		}
		facingRightOf := map[string]string{
			"south": "west", "north": "east", "east": "south", "west": "north",
		}

		for _, b := range blocks {
			if b.name != "minecraft:spruce_stairs" {
				continue
			}
			facing := b.props["facing"]
			fwd := facingVec[facing]

			front := get(b.x+fwd.x, b.y, b.z+fwd.z)
			back := get(b.x-fwd.x, b.y, b.z-fwd.z)

			// Inner corner
			if back != nil && back.name == "minecraft:spruce_stairs" && back.props["half"] == b.props["half"] {
				bf := back.props["facing"]
				if bf == facingLeftOf[facing] {
					b.props["shape"] = "inner_left"
					continue
				}
				if bf == facingRightOf[facing] {
					b.props["shape"] = "inner_right"
					continue
				}
			}
			// Outer corner
			if front != nil && front.name == "minecraft:spruce_stairs" && front.props["half"] == b.props["half"] {
				ff := front.props["facing"]
				if ff == facingLeftOf[facing] {
					b.props["shape"] = "outer_left"
					continue
				}
				if ff == facingRightOf[facing] {
					b.props["shape"] = "outer_right"
					continue
				}
			}
		}
	}

	// --- Normalize: shift so min = 0
	minX, minY, minZ := math.MaxInt32, math.MaxInt32, math.MaxInt32
	maxX, maxY, maxZ := math.MinInt32, math.MinInt32, math.MinInt32
	for _, b := range blocks {
		if b.x < minX {
			minX = b.x
		}
		if b.y < minY {
			minY = b.y
		}
		if b.z < minZ {
			minZ = b.z
		}
		if b.x > maxX {
			maxX = b.x
		}
		if b.y > maxY {
			maxY = b.y
		}
		if b.z > maxZ {
			maxZ = b.z
		}
	}
	if len(blocks) == 0 {
		minX, minY, minZ = 0, 0, 0
		maxX, maxY, maxZ = 0, 0, 0
	}

	// Map block names to block types
	nameToType := map[string]int{
		"minecraft:spruce_planks":   BlockPlank,
		"minecraft:spruce_slab":     BlockSlabBot,
		"minecraft:spruce_stairs":   BlockStair,
		"minecraft:spruce_fence":    BlockFence,
		"minecraft:spruce_trapdoor": BlockTrapdoor,
	}

	var result []Block
	for _, b := range blocks {
		bt, ok := nameToType[b.name]
		if !ok {
			bt = BlockPlank
		}
		result = append(result, Block{
			X:     b.x - minX,
			Y:     b.y - minY,
			Z:     b.z - minZ,
			Type:  bt,
			Props: b.props,
		})
	}

	return &GenerateResult{
		Blocks: result,
		SizeX:  maxX - minX + 1,
		SizeY:  maxY - minY + 1,
		SizeZ:  maxZ - minZ + 1,
		Materials: MaterialConfig{
			WoodType: p.WoodType,
		},
	}, nil
}
