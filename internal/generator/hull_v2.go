package generator

import (
	"math"
	"sort"
)

// generateHullV2 builds a hull from lofted geometry instead of v1's
// separable width product. Three ideas drive the shape:
//
//  1. Profile curves: the stem leans forward with height (StemRake/StemCurve)
//     and the stern leans aft (SternRake), so silhouettes read as ships
//     instead of rectangles. The keel gains rocker.
//  2. Section lofting: cross-section shape morphs along the length — V-shaped
//     entry sections at the bow (BowSectionV), full superellipse sections
//     midship (MidFullness, Deadrise), flatter sections aft (SternFullness).
//  3. Slope-aware surfacing: stairs (both halves, so overhanging flare and
//     counters work) and deck slabs are chosen from the width/height
//     gradients everywhere, not only in special-cased regions.
//
// The JS engine in template/static/generators.js mirrors this function; keep
// the two in sync (cross-checked by testdata fixtures).
func generateHullV2(p HullParams) (*GenerateResult, error) {
	L := p.Length
	D := p.Depth
	length := float64(L)
	depth := float64(D)
	halfBeam := float64(p.Beam) / 2

	smootherstep := func(t float64) float64 {
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return t * t * t * (t*(t*6-15) + 10)
	}
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }

	// --- Profile curves -----------------------------------------------------

	// Stem: how far (in blocks) the hull end at height yNorm sits behind the
	// tip at deck level. 0 at deck, max at keel. StemCurve bends the profile:
	// negative = concave clipper stem, positive = convex spoon bow.
	stemSetbackMax := p.StemRake * depth
	stemSetbackAt := func(yNorm float64) float64 {
		t := 1 - clamp01(yNorm)
		shape := t
		if p.StemCurve > 0 {
			shape = lerp(t, t*t, p.StemCurve)
		} else if p.StemCurve < 0 {
			shape = lerp(t, math.Sqrt(t), -p.StemCurve)
		}
		return stemSetbackMax * shape
	}

	// Stern: same idea aft. DoubleEnder mirrors the stem instead.
	sternSetbackMax := p.SternRake * depth
	sternSetbackAt := func(yNorm float64) float64 {
		if p.DoubleEnder {
			return stemSetbackAt(yNorm)
		}
		t := 1 - clamp01(yNorm)
		return sternSetbackMax * t
	}

	// Keel line: rocker curves the whole keel up toward the ends; the v1
	// bow/stern keel rises still apply on top (max wins).
	keelYAtF := func(zNorm float64) float64 {
		rise := 0.0
		if p.Rocker > 0 {
			t := math.Abs(zNorm-0.5) * 2
			rise = p.Rocker * depth * math.Pow(t, p.KeelCurve)
		}
		if p.BowKeelRise > 0 && p.BowKeelLength > 0 {
			start := 1.0 - float64(p.BowKeelLength)/length
			if zNorm > start {
				t := (zNorm - start) / math.Max(1-start, 0.001)
				r := math.Pow(t, p.KeelCurve) * p.BowKeelRise * depth
				if r > rise {
					rise = r
				}
			}
		}
		sternRiseP, sternLenP := p.SternKeelRise, p.SternKeelLength
		if p.DoubleEnder {
			sternRiseP, sternLenP = p.BowKeelRise, p.BowKeelLength
		}
		if sternRiseP > 0 && sternLenP > 0 {
			end := float64(sternLenP) / length
			if zNorm < end {
				t := (end - zNorm) / math.Max(end, 0.001)
				r := math.Pow(t, p.KeelCurve) * sternRiseP * depth
				if r > rise {
					rise = r
				}
			}
		}
		return rise
	}

	// --- Plan curve (deck outline) ------------------------------------------

	// Fair plan curve with a parallel midbody: width is 1 inside the midbody,
	// then eases to 0 through the entrance (bow) and run (stern).
	bowLen := float64(p.BowLength)
	sternLen := float64(p.SternLength)
	if p.DoubleEnder {
		sternLen = bowLen
	}
	midLo := sternLen / length
	midHi := 1 - bowLen/length
	// ParallelMidbody expands the full-width band around its center.
	midCenter := lerp(midLo, midHi, 0.5-p.MidWidthBias*0.35)
	pmHalf := p.ParallelMidbody / 2
	fullLo := math.Max(midLo, midCenter-pmHalf)
	fullHi := math.Min(midHi, midCenter+pmHalf)

	// The blend segments between the taper regions and the ParallelMidbody
	// band start well below full beam (0.78/0.84) so the waterline is a
	// continuous curve instead of taper-flat-taper.
	planAt := func(zNorm, yNorm float64) float64 {
		switch {
		case zNorm < midLo: // run (stern taper)
			t := zNorm / math.Max(midLo, 0.001)
			st := smootherstep(t)
			if p.DoubleEnder {
				return math.Pow(st, p.BowSharpness)
			}
			switch p.SternStyle {
			case "square":
				// The flat transom exists above the waterline only; the
				// run tapers underneath like a round stern.
				f := math.Pow(st, p.SternSharpness)
				floor := lerp(0.12, 0.72, smootherstep(clamp01(yNorm)))
				if f < floor {
					f = floor
				}
				return f
			case "round":
				return math.Pow(st, p.SternSharpness*0.55)
			default: // pointed
				return math.Pow(st, p.SternSharpness)
			}
		case zNorm > midHi: // entrance (bow taper)
			t := (1 - zNorm) / math.Max(1-midHi, 0.001)
			st := smootherstep(t)
			base := math.Pow(st, p.BowSharpness)
			if p.BowCurve > 0 {
				base = lerp(base, math.Sqrt(st), p.BowCurve)
			} else if p.BowCurve < 0 {
				base = lerp(base, st*st*st, -p.BowCurve)
			}
			return base
		case zNorm >= fullLo && zNorm <= fullHi:
			return 1
		case zNorm < fullLo: // fair curve from run into the midbody
			t := (zNorm - midLo) / math.Max(fullLo-midLo, 0.001)
			return lerp(0.78, 1, smootherstep(t))
		default: // fair curve from midbody into the entrance
			t := (midHi - zNorm) / math.Max(midHi-fullHi, 0.001)
			return lerp(0.84, 1, smootherstep(t))
		}
	}

	// --- Section lofting ------------------------------------------------------

	// Section shape exponent k for w = sin(yc*pi/2)^k: k ~0.55 is a full,
	// boxy section; k ~1.6 is a sharp V. The sine base makes topsides
	// approach vertical at the deck while the bottom stays curved — the
	// wineglass profile of real hulls. Deadrise sharpens the bottom;
	// BowSectionV / SternFullness morph k along the length.
	fullnessToK := func(f float64) float64 { return lerp(1.6, 0.55, clamp01(f)) }
	sectionKAt := func(zNorm float64) float64 {
		midK := fullnessToK(p.MidFullness)
		bowK := midK + p.BowSectionV*0.9
		sternK := fullnessToK(lerp(p.SternFullness, p.MidFullness, 0.35))
		if p.DoubleEnder {
			sternK = bowK
		}
		switch {
		case zNorm > midHi:
			t := (zNorm - midHi) / math.Max(1-midHi, 0.001)
			return lerp(midK, bowK, smootherstep(t))
		case zNorm < midLo:
			t := (midLo - zNorm) / math.Max(midLo, 0.001)
			return lerp(midK, sternK, smootherstep(t))
		default:
			return midK
		}
	}

	// Cross-section half-width factor at height yNorm (0 keel .. 1 deck).
	// The keel flat is a fraction of v1's BottomPinch — a keel is a spine,
	// not a barge floor. Flare and tumblehome adjust the topsides;
	// above-deck (yNorm > 1) tapers like v1 castle sides.
	keelHalf := p.BottomPinch * 0.25
	sectionAt := func(yNorm, k float64) float64 {
		yc := clamp01(yNorm)
		body := math.Pow(math.Sin(yc*math.Pi/2), k+p.Deadrise*0.8)
		base := keelHalf + (1-keelHalf)*body
		flare := p.HullFlare * math.Pow(yc, p.FlareCurve)
		tumble := p.Tumblehome * math.Pow(yc, p.TumbleCurve)
		above := yNorm - 1
		if above < 0 {
			above = 0
		}
		castleTaper := above*0.32 + above*above*0.18
		r := base + flare - tumble - castleTaper
		if r < 0.06 {
			r = 0.06
		}
		return r
	}

	// --- Deck line ------------------------------------------------------------

	deckYAtF := func(z int) float64 {
		y := depth
		zNorm := float64(z) / math.Max(length-1, 1)
		if p.SheerCurve > 0 {
			t := math.Abs(zNorm-0.5) * 2
			y += p.SheerCurve * depth * math.Pow(t, p.SheerCurveExp)
		}
		if p.CastleHeight > 0 && p.CastleLength > 0 {
			cL := p.CastleLength
			blend := p.CastleBlend
			if b := int(float64(cL) * 0.55); b < blend {
				blend = b
			}
			if blend < 2 {
				blend = 2
			}
			if z < cL-blend {
				y += float64(p.CastleHeight)
			} else if z < cL {
				t := float64(z-(cL-blend)) / float64(blend)
				y += float64(p.CastleHeight) * (1 - smootherstep(t))
			}
		}
		if p.ForecastleHeight > 0 && p.ForecastleLength > 0 {
			fL := p.ForecastleLength
			blend := p.CastleBlend
			if b := int(float64(fL) * 0.55); b < blend {
				blend = b
			}
			if blend < 2 {
				blend = 2
			}
			zFromBow := L - 1 - z
			if zFromBow < fL-blend {
				y += float64(p.ForecastleHeight)
			} else if zFromBow < fL {
				t := float64(zFromBow-(fL-blend)) / float64(blend)
				y += float64(p.ForecastleHeight) * (1 - smootherstep(t))
			}
		}
		return y
	}

	// --- Build the half-width field -------------------------------------------

	maxDeckY := D
	deckYArr := make([]int, L)
	keelYArr := make([]int, L)
	for z := 0; z < L; z++ {
		deckYArr[z] = int(math.Round(deckYAtF(z)))
		if deckYArr[z] > maxDeckY {
			maxDeckY = deckYArr[z]
		}
	}
	topY := maxDeckY
	if p.ClosedHull {
		topY = 2 * D
	}

	// hw[y][z]: negative = no hull at this height/position.
	hw := make([][]float64, topY+1)
	for y := range hw {
		hw[y] = make([]float64, L)
		for z := range hw[y] {
			hw[y][z] = -1
		}
	}

	for z := 0; z < L; z++ {
		zNormBase := float64(z) / math.Max(length-1, 1)
		keelYArr[z] = int(math.Round(keelYAtF(zNormBase)))
		colTop := deckYArr[z]
		if p.ClosedHull {
			colTop = 2 * D
		}
		for y := keelYArr[z]; y <= colTop && y <= topY; y++ {
			yNorm := float64(y) / math.Max(depth, 1)
			if p.ClosedHull && y > D {
				// Mirror the section above the widest point into a closed
				// envelope; no deck, castles or flare above.
				yNorm = float64(2*D-y) / math.Max(depth, 1)
				if yNorm < 0 {
					continue
				}
			}
			// Profile: shrink the usable length at this height by the stem
			// and stern setbacks, then evaluate the plan curve in the
			// shortened coordinate system.
			sb := sternSetbackAt(yNorm)
			st := stemSetbackAt(yNorm)
			zLo := sb
			zHi := length - 1 - st
			if zHi <= zLo {
				continue
			}
			zf := float64(z)
			if zf < zLo-0.5 || zf > zHi+0.5 {
				continue
			}
			zNorm := clamp01((zf - zLo) / (zHi - zLo))

			w := planAt(zNorm, yNorm) * sectionAt(yNorm, sectionKAt(zNorm)) * halfBeam
			if w < 0.15 {
				continue
			}
			hw[y][z] = w
		}
	}

	// Fair along Z everywhere (not just bow/stern): 3-tap weighted average.
	for y := 0; y <= topY; y++ {
		sm := make([]float64, L)
		copy(sm, hw[y])
		for z := 1; z < L-1; z++ {
			a, b, c := hw[y][z-1], hw[y][z], hw[y][z+1]
			if b < 0 {
				continue
			}
			if a < 0 {
				a = 0
			}
			if c < 0 {
				c = 0
			}
			sm[z] = a*0.25 + b*0.5 + c*0.25
		}
		hw[y] = sm
	}

	// --- Voxelize ---------------------------------------------------------------

	type coord [3]int
	inHull := make(map[coord]bool)
	hwInt := make([][]int, topY+1)
	for y := range hwInt {
		hwInt[y] = make([]int, L)
		for z := range hwInt[y] {
			hwInt[y][z] = -1
		}
	}
	for y := 0; y <= topY; y++ {
		for z := 0; z < L; z++ {
			w := hw[y][z]
			if w < 0.15 {
				continue
			}
			maxX := int(math.Max(0, math.Round(w-0.0001)))
			hwInt[y][z] = maxX
			for x := -maxX; x <= maxX; x++ {
				inHull[coord{x, y, z}] = true
			}
		}
	}
	has := func(x, y, z int) bool { return inHull[coord{x, y, z}] }
	hwAt := func(y, z int) int {
		if y < 0 || y > topY || z < 0 || z >= L {
			return -1
		}
		return hwInt[y][z]
	}

	// --- Shell + deck -------------------------------------------------------------

	type blockEntry struct {
		x, y, z int
		name    string
		props   map[string]string
	}
	blocks := make(map[coord]*blockEntry)
	set := func(x, y, z int, name string, props map[string]string) {
		blocks[coord{x, y, z}] = &blockEntry{x: x, y: y, z: z, name: name, props: props}
	}
	get := func(x, y, z int) *blockEntry { return blocks[coord{x, y, z}] }

	for k := range inHull {
		x, y, z := k[0], k[1], k[2]
		exposed := !has(x-1, y, z) || !has(x+1, y, z) ||
			!has(x, y-1, z) || !has(x, y+1, z) ||
			!has(x, y, z-1) || !has(x, y, z+1)
		isDeck := !p.ClosedHull && y == deckYArr[z]
		if exposed || isDeck {
			set(x, y, z, "planks", nil)
		}
	}

	// --- Slope-aware stairs ---------------------------------------------------------

	// Lateral: hull widening upward gets top-half stairs (v1 behavior);
	// hull narrowing upward (tumblehome, closed-hull top) gets bottom-half
	// stairs hanging under the wider row — v1 could not express overhangs.
	for z := 0; z < L; z++ {
		for y := 0; y < topY; y++ {
			hwHere := hwAt(y, z)
			hwUp := hwAt(y+1, z)
			if hwHere < 0 || hwUp < 0 {
				continue
			}
			if hwUp > hwHere {
				for xN := hwHere + 1; xN <= hwUp; xN++ {
					if has(xN, y, z) || get(xN, y, z) != nil {
						continue
					}
					if !has(xN, y+1, z) {
						continue
					}
					set(xN, y, z, "stairs", map[string]string{"facing": "east", "half": "top", "shape": "straight", "waterlogged": "false"})
					set(-xN, y, z, "stairs", map[string]string{"facing": "west", "half": "top", "shape": "straight", "waterlogged": "false"})
				}
			} else if hwUp < hwHere {
				for xN := hwUp + 1; xN <= hwHere; xN++ {
					if has(xN, y+1, z) || get(xN, y+1, z) != nil {
						continue
					}
					if !has(xN, y, z) {
						continue
					}
					set(xN, y+1, z, "stairs", map[string]string{"facing": "east", "half": "bottom", "shape": "straight", "waterlogged": "false"})
					set(-xN, y+1, z, "stairs", map[string]string{"facing": "west", "half": "bottom", "shape": "straight", "waterlogged": "false"})
				}
			}
		}
	}

	// Longitudinal taper stairs, both directions.
	placeLongStair := func(x, y, z int, facing string) {
		if has(x, y, z) || get(x, y, z) != nil {
			return
		}
		set(x, y, z, "stairs", map[string]string{"facing": facing, "half": "top", "shape": "straight", "waterlogged": "false"})
	}
	for y := 0; y <= topY; y++ {
		for z := 0; z < L; z++ {
			hwThis := hwAt(y, z)
			if hwThis < 0 {
				continue
			}
			if hwF := hwAt(y, z+1); hwF >= 0 && hwF < hwThis {
				for x := hwF + 1; x <= hwThis; x++ {
					placeLongStair(x, y, z+1, "south")
					if x != 0 {
						placeLongStair(-x, y, z+1, "south")
					}
				}
			}
			if hwB := hwAt(y, z-1); hwB >= 0 && hwB < hwThis && z > 0 {
				for x := hwB + 1; x <= hwThis; x++ {
					placeLongStair(x, y, z-1, "north")
					if x != 0 {
						placeLongStair(-x, y, z-1, "north")
					}
				}
			}
		}
	}

	// Keel/rocker steps: stairs under the rising keel line.
	for z := 0; z < L-1; z++ {
		k0, k1 := keelYArr[z], keelYArr[z+1]
		if k1 == k0 {
			continue
		}
		yFill, zFill, facing := k0-1, z, "north"
		refZ := z
		if k1 > k0 {
			yFill, zFill, facing = k1-1, z+1, "south"
			refZ = z + 1
		}
		hwRef := hwAt(yFill+1, refZ)
		if hwRef < 0 {
			continue
		}
		for x := -hwRef; x <= hwRef; x++ {
			if has(x, yFill, zFill) || get(x, yFill, zFill) != nil {
				continue
			}
			set(x, yFill, zFill, "stairs", map[string]string{"facing": facing, "half": "top", "shape": "straight", "waterlogged": "false"})
		}
	}

	// De-stack: a stair directly above a same-half stair reads as a wall;
	// keep the lower, replace the upper with planks. Deterministic order.
	{
		type se struct{ x, y, z int }
		var stairs []se
		for k, b := range blocks {
			if b.name == "stairs" {
				stairs = append(stairs, se{k[0], k[1], k[2]})
			}
		}
		sort.Slice(stairs, func(i, j int) bool {
			if stairs[i].y != stairs[j].y {
				return stairs[i].y > stairs[j].y
			}
			if stairs[i].z != stairs[j].z {
				return stairs[i].z < stairs[j].z
			}
			return stairs[i].x < stairs[j].x
		})
		for _, s := range stairs {
			b := get(s.x, s.y, s.z)
			below := get(s.x, s.y-1, s.z)
			if b != nil && b.name == "stairs" && below != nil && below.name == "stairs" && below.props["half"] == b.props["half"] {
				set(s.x, s.y, s.z, "planks", nil)
			}
		}
	}

	// Deck-sheer slabs: soften each 1-block deck step with a slab on the
	// lower side of the step.
	if !p.ClosedHull {
		for z := 0; z < L-1; z++ {
			d0, d1 := deckYArr[z], deckYArr[z+1]
			// Only soften exact 1-block steps; larger jumps (castle fronts)
			// read better as clean walls than as terraced slab aprons.
			if d1-d0 != 1 && d0-d1 != 1 {
				continue
			}
			zLow, dLow := z, d0
			if d1 < d0 {
				zLow, dLow = z+1, d1
			}
			hwHere := hwAt(dLow, zLow)
			if hwHere < 1 {
				continue
			}
			for x := -hwHere + 1; x <= hwHere-1; x++ {
				if get(x, dLow+1, zLow) == nil && !has(x, dLow+1, zLow) {
					set(x, dLow+1, zLow, "slab", map[string]string{"type": "bottom", "waterlogged": "false"})
				}
			}
		}
	}

	// Stem/stern posts: a rising post at the extreme ends
	// (longship/dragon-ship silhouettes), tapered with a stair on top.
	if !p.ClosedHull {
		placePost := func(fromBow bool, height int) {
			if height <= 0 {
				return
			}
			zStart, zEnd, step := L-1, -1, -1
			if !fromBow {
				zStart, zEnd, step = 0, L, 1
			}
			zPost := -1
			for z := zStart; z != zEnd; z += step {
				if hwAt(deckYArr[z], z) >= 0 {
					zPost = z
					break
				}
			}
			if zPost < 0 {
				return
			}
			deckY := deckYArr[zPost]
			for y := deckY + 1; y <= deckY+height; y++ {
				set(0, y, zPost, "planks", nil)
			}
			facing := "south"
			if !fromBow {
				facing = "north"
			}
			set(0, deckY+height+1, zPost, "stairs", map[string]string{"facing": facing, "half": "bottom", "shape": "straight", "waterlogged": "false"})
		}
		placePost(true, p.StemPostHeight)
		sternPost := p.SternPostHeight
		if p.DoubleEnder && sternPost == 0 {
			sternPost = p.StemPostHeight
		}
		placePost(false, sternPost)
	}

	// --- Furnishing (v1 semantics) ------------------------------------------------

	if p.HasWindows && p.CastleHeight >= 2 && p.CastleLength > 0 && !p.ClosedHull {
		z := 0
		wy := D + 1
		if deckYArr[z] > D {
			hwBack := hwAt(wy, z)
			if hwBack >= 1 {
				for x := -hwBack + 1; x <= hwBack-1; x += 2 {
					if b := get(x, wy, z); b != nil && b.name == "planks" {
						set(x, wy, z, "trapdoor", map[string]string{"facing": "north", "half": "bottom", "open": "true", "powered": "false", "waterlogged": "false"})
					}
				}
			}
		}
	}

	if !p.ClosedHull {
		for z := 0; z < L; z++ {
			deckY := deckYArr[z]
			hwD := hwAt(deckY, z)
			if hwD < 1 {
				continue
			}
			y := deckY + 1
			canInset := hwD >= 2
			switch {
			case p.HasTrim && p.HasRailings && canInset:
				if get(hwD, y, z) == nil && !has(hwD, y, z) {
					set(hwD, y, z, "slab", map[string]string{"type": "bottom", "waterlogged": "false"})
				}
				if get(-hwD, y, z) == nil && !has(-hwD, y, z) {
					set(-hwD, y, z, "slab", map[string]string{"type": "bottom", "waterlogged": "false"})
				}
				set(hwD-1, y, z, "fence", nil)
				if hwD-1 > 0 {
					set(-(hwD - 1), y, z, "fence", nil)
				}
			case p.HasTrim:
				if get(hwD, y, z) == nil && !has(hwD, y, z) {
					set(hwD, y, z, "slab", map[string]string{"type": "bottom", "waterlogged": "false"})
				}
				if hwD > 0 && get(-hwD, y, z) == nil && !has(-hwD, y, z) {
					set(-hwD, y, z, "slab", map[string]string{"type": "bottom", "waterlogged": "false"})
				}
			case p.HasRailings:
				set(hwD, y, z, "fence", nil)
				if hwD > 0 {
					set(-hwD, y, z, "fence", nil)
				}
			}
		}
		for _, b := range blocks {
			if b.name != "fence" {
				continue
			}
			if below := get(b.x, b.y-1, b.z); below != nil && below.name == "slab" {
				delete(blocks, coord{b.x, b.y - 1, b.z})
			}
		}
	}

	if p.HasGunPorts && p.GunPortRow > 0 && !p.ClosedHull {
		yPort := D - p.GunPortRow
		midKeel := keelYArr[L/2]
		if midKeel+1 > yPort {
			yPort = midKeel + 1
		}
		for z := 3; z < L-3; z += p.GunPortSpacing {
			hwP := hwAt(yPort, z)
			if hwP < 1 {
				continue
			}
			set(hwP, yPort, z, "trapdoor", map[string]string{"facing": "east", "half": "bottom", "open": "true", "powered": "false", "waterlogged": "false"})
			if hwP > 0 {
				set(-hwP, yPort, z, "trapdoor", map[string]string{"facing": "west", "half": "bottom", "open": "true", "powered": "false", "waterlogged": "false"})
			}
		}
	}

	// Cleanup: raked profiles can strand small chains of decorative blocks
	// with no path back to the hull. Keep only blocks 6-connected to the
	// hull volume (BFS seeded from every hull cell).
	{
		dirs := [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
		reached := make(map[coord]bool, len(blocks))
		queue := make([]coord, 0, len(inHull))
		for k := range inHull {
			queue = append(queue, k)
		}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			for _, d := range dirs {
				n := coord{c[0] + d[0], c[1] + d[1], c[2] + d[2]}
				if reached[n] || inHull[n] {
					continue
				}
				if blocks[n] != nil {
					reached[n] = true
					queue = append(queue, n)
				}
			}
		}
		for k := range blocks {
			if !inHull[k] && !reached[k] {
				delete(blocks, k)
			}
		}
	}

	// Fence connection states
	for _, b := range blocks {
		if b.name != "fence" {
			continue
		}
		conn := func(dx, dz int) string {
			n := get(b.x+dx, b.y, b.z+dz)
			if n == nil {
				return "false"
			}
			if n.name == "fence" || n.name == "planks" || n.name == "stairs" {
				return "true"
			}
			return "false"
		}
		b.props = map[string]string{
			"east": conn(1, 0), "west": conn(-1, 0),
			"south": conn(0, 1), "north": conn(0, -1),
			"waterlogged": "false",
		}
	}

	// Stair corner shapes (same rules as v1)
	{
		type vec2 struct{ x, z int }
		facingVec := map[string]vec2{"south": {0, 1}, "north": {0, -1}, "east": {1, 0}, "west": {-1, 0}}
		leftOf := map[string]string{"south": "east", "north": "west", "east": "north", "west": "south"}
		rightOf := map[string]string{"south": "west", "north": "east", "east": "south", "west": "north"}
		for _, b := range blocks {
			if b.name != "stairs" {
				continue
			}
			f := b.props["facing"]
			fwd := facingVec[f]
			front := get(b.x+fwd.x, b.y, b.z+fwd.z)
			back := get(b.x-fwd.x, b.y, b.z-fwd.z)
			if back != nil && back.name == "stairs" && back.props["half"] == b.props["half"] {
				if back.props["facing"] == leftOf[f] {
					b.props["shape"] = "inner_left"
					continue
				}
				if back.props["facing"] == rightOf[f] {
					b.props["shape"] = "inner_right"
					continue
				}
			}
			if front != nil && front.name == "stairs" && front.props["half"] == b.props["half"] {
				if front.props["facing"] == leftOf[f] {
					b.props["shape"] = "outer_left"
					continue
				}
				if front.props["facing"] == rightOf[f] {
					b.props["shape"] = "outer_right"
					continue
				}
			}
		}
	}

	// --- Emit ---------------------------------------------------------------------

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
		minX, minY, minZ, maxX, maxY, maxZ = 0, 0, 0, 0, 0, 0
	}

	nameToType := map[string]int{
		"planks":   BlockPlank,
		"slab":     BlockSlabBot,
		"stairs":   BlockStair,
		"fence":    BlockFence,
		"trapdoor": BlockTrapdoor,
	}
	var result []Block
	for _, b := range blocks {
		bt, ok := nameToType[b.name]
		if !ok {
			bt = BlockPlank
		}
		result = append(result, Block{X: b.x - minX, Y: b.y - minY, Z: b.z - minZ, Type: bt, Props: b.props})
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
