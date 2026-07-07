package generator

import (
	"math"
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

	// --- Continuous hull test ---------------------------------------------------

	// deckYAtFloat is the deck surface height at a continuous z coordinate;
	// keeping it continuous lets the shape fitter express the sheer curve
	// with slabs instead of hard one-block cliffs.
	deckYAtFloat := func(zf float64) float64 {
		y := depth
		zNorm := zf / math.Max(length-1, 1)
		if p.SheerCurve > 0 {
			t := math.Abs(zNorm-0.5) * 2
			y += p.SheerCurve * depth * math.Pow(t, p.SheerCurveExp)
		}
		if p.CastleHeight > 0 && p.CastleLength > 0 {
			cL := float64(p.CastleLength)
			blend := float64(p.CastleBlend)
			if b := cL * 0.55; b < blend {
				blend = b
			}
			if blend < 2 {
				blend = 2
			}
			if zf < cL-blend {
				y += float64(p.CastleHeight)
			} else if zf < cL {
				t := (zf - (cL - blend)) / blend
				y += float64(p.CastleHeight) * (1 - smootherstep(t))
			}
		}
		if p.ForecastleHeight > 0 && p.ForecastleLength > 0 {
			fL := float64(p.ForecastleLength)
			blend := float64(p.CastleBlend)
			if b := fL * 0.55; b < blend {
				blend = b
			}
			if blend < 2 {
				blend = 2
			}
			zFromBow := length - 1 - zf
			if zFromBow < fL-blend {
				y += float64(p.ForecastleHeight)
			} else if zFromBow < fL {
				t := (zFromBow - (fL - blend)) / blend
				y += float64(p.ForecastleHeight) * (1 - smootherstep(t))
			}
		}
		return y
	}

	// insideAt is the continuous hull volume test. All quantization happens
	// later, in the shape fitter — never here.
	insideAt := func(xs, ys, zs float64) bool {
		if zs < -0.49 || zs > length-0.51 {
			return false
		}
		zNormBase := zs / math.Max(length-1, 1)
		if zNormBase < 0 {
			zNormBase = 0
		}
		if zNormBase > 1 {
			zNormBase = 1
		}
		keelY := keelYAtF(zNormBase)
		if ys < keelY {
			return false
		}
		// Sections are lofted between the keel line and the deck: remapping
		// (rather than chopping at the keel plane) keeps the underside a
		// continuous curve where the keel rises toward the ends.
		bottomSpan := depth - keelY
		if bottomSpan < 1 {
			bottomSpan = 1
		}
		var yNorm float64
		if p.ClosedHull {
			if ys > 2*depth {
				return false
			}
			if ys <= depth {
				yNorm = (ys - keelY) / bottomSpan
			} else {
				yNorm = (2*depth - ys) / depth
			}
			if yNorm < 0 {
				return false
			}
		} else {
			if ys > deckYAtFloat(zs) {
				return false
			}
			yNorm = (ys - keelY) / bottomSpan
		}
		sb := sternSetbackAt(yNorm)
		st := stemSetbackAt(yNorm)
		zLo, zHi := sb, length-1-st
		if zHi <= zLo || zs < zLo || zs > zHi {
			return false
		}
		zN := clamp01((zs - zLo) / (zHi - zLo))
		w := planAt(zN, yNorm) * sectionAt(yNorm, sectionKAt(zN)) * halfBeam
		if w < 0.15 {
			return false
		}
		return math.Abs(xs) <= w
	}

	// --- Grid extents -------------------------------------------------------------

	maxDeckY := D
	deckYArr := make([]int, L)
	keelYArr := make([]int, L)
	for z := 0; z < L; z++ {
		deckYArr[z] = int(math.Round(deckYAtFloat(float64(z))))
		if deckYArr[z] > maxDeckY {
			maxDeckY = deckYArr[z]
		}
		keelYArr[z] = int(math.Round(keelYAtF(float64(z) / math.Max(length-1, 1))))
	}
	topY := maxDeckY
	if p.ClosedHull {
		topY = 2 * D
	}
	xMax := int(math.Ceil(halfBeam*(1+p.HullFlare+p.SternOverhang))) + 2

	// --- Occupancy: corner grid + per-cell classification --------------------------

	// Cell (x,y,z) spans a unit cube centred on integer coordinates. Corners
	// live on the half-integer lattice and are shared between cells, so one
	// insideAt call serves eight cells.
	nx, ny, nz := 2*xMax+2, topY+2, L+1
	corner := make([]bool, nx*ny*nz)
	cIdx := func(i, j, k int) int { return (k*ny+j)*nx + i }
	for k := 0; k < nz; k++ {
		for j := 0; j < ny; j++ {
			for i := 0; i < nx; i++ {
				corner[cIdx(i, j, k)] = insideAt(float64(i-xMax)-0.5, float64(j)-0.5, float64(k)-0.5)
			}
		}
	}

	// solid = all eight corners inside; empty = none (plus centre check for
	// thin features). Everything else is a surface cell for the fitter.
	const (
		cellEmpty = 0
		cellSolid = 1
		cellSurf  = 2
	)
	cell := make([]uint8, (2*xMax+1)*(topY+1)*L)
	cellAt := func(x, y, z int) int {
		if x < -xMax || x > xMax || y < 0 || y > topY || z < 0 || z >= L {
			return cellEmpty
		}
		return int(cell[(z*(topY+1)+y)*(2*xMax+1)+(x+xMax)])
	}
	setCell := func(x, y, z int, v uint8) {
		cell[(z*(topY+1)+y)*(2*xMax+1)+(x+xMax)] = v
	}
	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			for x := -xMax; x <= xMax; x++ {
				in := 0
				for dk := 0; dk <= 1; dk++ {
					for dj := 0; dj <= 1; dj++ {
						for di := 0; di <= 1; di++ {
							if corner[cIdx(x+xMax+di, y+dj, z+dk)] {
								in++
							}
						}
					}
				}
				switch {
				case in == 8:
					setCell(x, y, z, cellSolid)
				case in == 0:
					if insideAt(float64(x), float64(y), float64(z)) {
						setCell(x, y, z, cellSurf)
					}
				default:
					setCell(x, y, z, cellSurf)
				}
			}
		}
	}

	// --- Shape fitting ---------------------------------------------------------------

	// Candidate shapes are tested against a 3x3x3 occupancy sample of the
	// cell. Small penalties bias ties toward solid blocks (watertight shells)
	// and strongly against air (no pinholes).
	type shapeCand struct {
		name    string
		props   map[string]string
		test    func(sx, sy, sz float64) bool
		penalty float64
	}
	sideEast := func(sx, sy, sz float64) bool { return sx >= 0 }
	sideWest := func(sx, sy, sz float64) bool { return sx <= 0 }
	sideSouth := func(sx, sy, sz float64) bool { return sz >= 0 }
	sideNorth := func(sx, sy, sz float64) bool { return sz <= 0 }
	stairCand := func(facing string, side func(float64, float64, float64) bool, half string) shapeCand {
		return shapeCand{
			name:  "stairs",
			props: map[string]string{"facing": facing, "half": half, "shape": "straight", "waterlogged": "false"},
			test: func(sx, sy, sz float64) bool {
				if half == "bottom" {
					return sy <= 0 || side(sx, sy, sz)
				}
				return sy >= 0 || side(sx, sy, sz)
			},
			penalty: 0.6,
		}
	}
	candidates := []shapeCand{
		{name: "", test: func(sx, sy, sz float64) bool { return false }, penalty: 3.0}, // air
		{name: "planks", test: func(sx, sy, sz float64) bool { return true }, penalty: 0.0},
		{name: "slab", props: map[string]string{"type": "bottom", "waterlogged": "false"},
			test: func(sx, sy, sz float64) bool { return sy <= 0 }, penalty: 0.9},
		{name: "slab_top", props: map[string]string{"type": "top", "waterlogged": "false"},
			test: func(sx, sy, sz float64) bool { return sy >= 0 }, penalty: 0.9},
		stairCand("east", sideEast, "bottom"),
		stairCand("west", sideWest, "bottom"),
		stairCand("north", sideNorth, "bottom"),
		stairCand("south", sideSouth, "bottom"),
		stairCand("east", sideEast, "top"),
		stairCand("west", sideWest, "top"),
		stairCand("north", sideNorth, "top"),
		stairCand("south", sideSouth, "top"),
	}
	var sampleOffsets [27][3]float64
	{
		vals := [3]float64{-1.0 / 3.0, 0, 1.0 / 3.0}
		n := 0
		for _, ox := range vals {
			for _, oy := range vals {
				for _, oz := range vals {
					sampleOffsets[n] = [3]float64{ox, oy, oz}
					n++
				}
			}
		}
	}

	type coord [3]int
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

	copyProps := func(m map[string]string) map[string]string {
		if m == nil {
			return nil
		}
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}

	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			for x := -xMax; x <= xMax; x++ {
				if cellAt(x, y, z) != cellSurf {
					continue
				}
				var occ [27]bool
				occCount := 0
				for s := 0; s < 27; s++ {
					o := sampleOffsets[s]
					if insideAt(float64(x)+o[0], float64(y)+o[1], float64(z)+o[2]) {
						occ[s] = true
						occCount++
					}
				}
				if occCount == 0 {
					continue
				}
				bestIdx, bestErr := 0, math.MaxFloat64
				for ci, cand := range candidates {
					errSum := cand.penalty
					for s := 0; s < 27; s++ {
						o := sampleOffsets[s]
						if cand.test(o[0], o[1], o[2]) != occ[s] {
							errSum++
						}
					}
					if errSum < bestErr {
						bestErr = errSum
						bestIdx = ci
					}
				}
				if candidates[bestIdx].name == "" {
					continue
				}
				set(x, y, z, candidates[bestIdx].name, copyProps(candidates[bestIdx].props))
			}
		}
	}

	// --- Shell + deck -----------------------------------------------------------------

	// Solid cells: keep only the shell (a face exposed to non-solid) plus the
	// deck row, mirroring v1's hollow hulls.
	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			for x := -xMax; x <= xMax; x++ {
				if cellAt(x, y, z) != cellSolid {
					continue
				}
				exposed := cellAt(x-1, y, z) != cellSolid || cellAt(x+1, y, z) != cellSolid ||
					cellAt(x, y-1, z) != cellSolid || cellAt(x, y+1, z) != cellSolid ||
					cellAt(x, y, z-1) != cellSolid || cellAt(x, y, z+1) != cellSolid
				isDeck := !p.ClosedHull && y == deckYArr[z]
				if exposed || isDeck {
					set(x, y, z, "planks", nil)
				}
			}
		}
	}

	// hwAt reports the outermost occupied |x| at a row, used by the
	// decoration passes (railings, gun ports, posts).
	hwAt := func(y, z int) int {
		if y < 0 || y > topY || z < 0 || z >= L {
			return -1
		}
		for x := xMax; x >= 0; x-- {
			c := cellAt(x, y, z)
			if c == cellSolid {
				return x
			}
			if c == cellSurf && get(x, y, z) != nil {
				return x
			}
		}
		return -1
	}
	has := func(x, y, z int) bool { return cellAt(x, y, z) == cellSolid }
	// inHull seeds the connectivity cleanup.
	inHull := make(map[coord]bool)
	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			for x := -xMax; x <= xMax; x++ {
				if cellAt(x, y, z) == cellSolid {
					inHull[coord{x, y, z}] = true
				}
			}
		}
	}


	// --- Stem/stern posts (kept from pass pipeline) --------------------------------
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
		"slab_top": BlockSlabTop,
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
