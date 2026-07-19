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
//  3. Column-quantized surfacing: the continuous half-width is rounded per
//     (y,z) row like v1, faired to remove single-block quantization spikes,
//     and stairs are placed exactly at the integer steps. Per-cell shape
//     fitting (the old marching-cubes-style pass) produced pockmarked walls
//     and floating slabs on large hulls; rounding whole rows keeps every
//     surface line coherent.
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

	// Sweep bends the whole hull — keel and deck together — up toward the
	// ends, applied as a vertical shear so sections keep their shape. This
	// is the lengthwise counterpart of the sideways plan curve.
	sweepAt := func(zNorm float64) float64 {
		if p.Sweep <= 0 {
			return 0
		}
		t := math.Abs(zNorm-0.5) * 2
		return p.Sweep * depth * t * t
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

	// --- Grid extents -------------------------------------------------------------

	maxDeckY := D
	deckYArr := make([]int, L)
	keelYArr := make([]int, L)
	for z := 0; z < L; z++ {
		zn := float64(z) / math.Max(length-1, 1)
		sw := sweepAt(zn)
		deckYArr[z] = int(math.Round(deckYAtFloat(float64(z)) + sw))
		if deckYArr[z] > maxDeckY {
			maxDeckY = deckYArr[z]
		}
		keelYArr[z] = int(math.Round(keelYAtF(zn) + sw))
	}
	topY := maxDeckY
	if p.ClosedHull {
		topY = 2*D + int(math.Round(sweepAt(0)))
	}

	// --- Column-quantized half-widths ----------------------------------------------

	// hwRowAt is the continuous half-width of the hull at integer row (y,z),
	// or -1 when the row is outside the hull. Quantization happens on whole
	// rows — never per cell — so every surface line stays coherent.
	hwRowAt := func(y, z int) float64 {
		if z < 0 || z >= L || y < 0 || y > topY {
			return -1
		}
		if y < keelYArr[z] {
			return -1
		}
		if !p.ClosedHull && y > deckYArr[z] {
			return -1
		}
		ys, zs := float64(y), float64(z)
		zNormBase := clamp01(zs / math.Max(length-1, 1))
		// Undo the sweep shear so the loft below sees an unbent hull.
		ys -= sweepAt(zNormBase)
		keelY := keelYAtF(zNormBase)
		bottomSpan := depth - keelY
		if bottomSpan < 1 {
			bottomSpan = 1
		}
		// The rounded keel and deck rows clamp to the continuous heights so
		// the rows the grid keeps get real loft widths.
		var yNorm float64
		if p.ClosedHull {
			if ys < keelY {
				ys = keelY
			}
			if ys <= depth {
				yNorm = (ys - keelY) / bottomSpan
			} else {
				yNorm = (2*depth - ys) / depth
			}
			if yNorm < 0 {
				return -1
			}
		} else {
			if d := deckYAtFloat(zs); ys > d {
				ys = d
			}
			if ys < keelY {
				ys = keelY
			}
			yNorm = (ys - keelY) / bottomSpan
		}
		sb := sternSetbackAt(yNorm)
		st := stemSetbackAt(yNorm)
		zLo, zHi := sb, length-1-st
		if zHi <= zLo || zs < zLo || zs > zHi {
			return -1
		}
		zN := clamp01((zs - zLo) / (zHi - zLo))
		w := planAt(zN, yNorm) * sectionAt(yNorm, sectionKAt(zN)) * halfBeam
		if w < 0.15 {
			return -1
		}
		return w
	}

	rawHW := make([][]float64, topY+1)
	for y := range rawHW {
		rawHW[y] = make([]float64, L)
		for z := range rawHW[y] {
			rawHW[y][z] = hwRowAt(y, z)
		}
	}

	// Fair the entrance and run along z (v1's smoothing): quantization noise
	// is most visible where the plan curve changes fastest.
	bowStartZ := L - p.BowLength
	sternEndZ := p.SternLength
	if p.DoubleEnder {
		sternEndZ = p.BowLength
	}
	for y := 0; y <= topY; y++ {
		sm := make([]float64, L)
		copy(sm, rawHW[y])
		for z := 1; z < L-1; z++ {
			if z >= sternEndZ && z <= bowStartZ {
				continue
			}
			prev, cur, next := rawHW[y][z-1], rawHW[y][z], rawHW[y][z+1]
			if cur < 0 {
				continue
			}
			if prev < 0 {
				prev = cur
			}
			if next < 0 {
				next = cur
			}
			sm[z] = prev*0.25 + cur*0.5 + next*0.25
		}
		rawHW[y] = sm
	}

	// Quantize rows, then remove single-row spikes: a lone ±1 outlier between
	// equal neighbours (along z or along y) is rounding noise, not shape.
	hwArr := make([][]int, topY+1)
	for y := range hwArr {
		hwArr[y] = make([]int, L)
		for z := range hwArr[y] {
			if rawHW[y][z] < 0 {
				hwArr[y][z] = -1
			} else {
				q := int(math.Round(rawHW[y][z] - 0.0001))
				if q < 0 {
					q = 0
				}
				hwArr[y][z] = q
			}
		}
	}
	for y := 0; y <= topY; y++ {
		for z := 1; z < L-1; z++ {
			a, b, c := hwArr[y][z-1], hwArr[y][z], hwArr[y][z+1]
			if a >= 0 && b >= 0 && a == c && (b == a+1 || b == a-1) {
				hwArr[y][z] = a
			}
		}
	}
	for z := 0; z < L; z++ {
		for y := 1; y < topY; y++ {
			a, b, c := hwArr[y-1][z], hwArr[y][z], hwArr[y+1][z]
			if a >= 0 && b >= 0 && a == c && (b == a+1 || b == a-1) {
				hwArr[y][z] = a
			}
		}
	}

	hasHull := func(x, y, z int) bool {
		if y < 0 || y > topY || z < 0 || z >= L {
			return false
		}
		hw := hwArr[y][z]
		return hw >= 0 && x >= -hw && x <= hw
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

	// --- Shell + deck -----------------------------------------------------------------

	// Keep only the shell (a face exposed to non-hull) plus the deck row,
	// mirroring v1's hollow hulls.
	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			hw := hwArr[y][z]
			if hw < 0 {
				continue
			}
			for x := -hw; x <= hw; x++ {
				exposed := !hasHull(x-1, y, z) || !hasHull(x+1, y, z) ||
					!hasHull(x, y-1, z) || !hasHull(x, y+1, z) ||
					!hasHull(x, y, z-1) || !hasHull(x, y, z+1)
				isDeck := !p.ClosedHull && y == deckYArr[z]
				if exposed || isDeck {
					set(x, y, z, "planks", nil)
				}
			}
		}
	}

	// --- Step smoothing --------------------------------------------------------------

	// One chamfer rule covers flare underhangs, bow/stern rakes and keel
	// rises: an empty cell directly under hull that also touches hull on a
	// horizontal face gets a top-half stair. Requiring the horizontal
	// neighbour keeps stairs seated in real step corners — no teeth hanging
	// from a face above. Facing follows vanilla semantics (the stair's TALL
	// side is on the facing side, per models/block/stairs.json), so it
	// points TOWARD the supporting neighbour: the tall half continues the
	// hull mass and the open notch chamfers away down the curve. In the
	// bow/stern tapers fore-aft support wins so the stem reads as one
	// stepped line instead of mixed directions.
	maxHW := 0
	for y := 0; y <= topY; y++ {
		for z := 0; z < L; z++ {
			if hwArr[y][z] > maxHW {
				maxHW = hwArr[y][z]
			}
		}
	}
	// The side-profile zone covers the plan tapers plus the keel-rise
	// spans: everywhere the hull's side silhouette is a diagonal, lateral
	// chamfers read as square teeth poking below the rake line, so only
	// fore-aft chamfers are allowed there.
	inTaper := make([]bool, L)
	for z := 0; z < L; z++ {
		zn := float64(z) / math.Max(length-1, 1)
		inTaper[z] = zn < midLo || zn > midHi
		if p.BowKeelRise > 0 && z >= L-1-p.BowKeelLength {
			inTaper[z] = true
		}
		if p.SternKeelRise > 0 && z <= p.SternKeelLength {
			inTaper[z] = true
		}
	}
	chamferFacing := func(x, y, z int) string {
		n := hasHull(x, y, z-1)
		s := hasHull(x, y, z+1)
		w := hasHull(x-1, y, z)
		e := hasHull(x+1, y, z)
		// Lateral chamfers only count when the supporting hull is INBOARD
		// (toward the centerline): those are real flare underhangs on the
		// hull's outer surface. Outboard-supported cells sit inside the
		// V-groove between the entry sections of sharp bows and sterns;
		// a stair there projects a full-square silhouette through the
		// open centerline and reads as teeth under the rake line.
		lateral := ""
		if x > 0 && w {
			lateral = "west"
		} else if x < 0 && e {
			lateral = "east"
		}
		foreAft := ""
		if n {
			foreAft = "north"
		} else if s {
			foreAft = "south"
		}
		if inTaper[z] {
			return foreAft
		}
		if lateral != "" {
			return lateral
		}
		return foreAft
	}
	for z := 0; z < L; z++ {
		for y := 0; y < topY; y++ {
			for x := -maxHW - 1; x <= maxHW+1; x++ {
				if hasHull(x, y, z) || !hasHull(x, y+1, z) {
					continue
				}
				if get(x, y, z) != nil {
					continue
				}
				f := chamferFacing(x, y, z)
				if f == "" {
					continue
				}
				set(x, y, z, "stairs", map[string]string{"facing": f, "half": "top", "shape": "straight", "waterlogged": "false"})
			}
		}
	}

	// Ledge caps: an empty cell directly on top of hull with horizontal hull
	// support gets a bottom-half stair, smoothing tumblehome ledges, castle
	// walls and the deck breaks a strong sweep or sheer produces. Above the
	// column's own deck only fore-aft caps are allowed — the gunwale edge
	// belongs to trim and railings.
	for z := 0; z < L; z++ {
		for y := 1; y <= topY; y++ {
			for x := -maxHW - 1; x <= maxHW+1; x++ {
				if hasHull(x, y, z) || !hasHull(x, y-1, z) {
					continue
				}
				if get(x, y, z) != nil {
					continue
				}
				aboveDeck := !p.ClosedHull && y > deckYArr[z]
				f := chamferFacing(x, y, z)
				if f == "" {
					continue
				}
				if aboveDeck && f != "south" && f != "north" {
					continue
				}
				set(x, y, z, "stairs", map[string]string{"facing": f, "half": "bottom", "shape": "straight", "waterlogged": "false"})
			}
		}
	}

	// De-stack: runs of same-facing same-half stairs on steep surfaces read
	// as serrated walls; keep the lowest stair of each run and plank the rest
	// (v1's proven rule). Deterministic order.
	{
		type se struct{ x, y, z int }
		var stairs []se
		for k, b := range blocks {
			if b.name == "stairs" {
				stairs = append(stairs, se{k[0], k[1], k[2]})
			}
		}
		sortStairs := func(a, b se) bool {
			if a.y != b.y {
				return a.y > b.y
			}
			if a.z != b.z {
				return a.z < b.z
			}
			return a.x < b.x
		}
		for i := 1; i < len(stairs); i++ {
			for j := i; j > 0 && sortStairs(stairs[j], stairs[j-1]); j-- {
				stairs[j], stairs[j-1] = stairs[j-1], stairs[j]
			}
		}
		for _, s := range stairs {
			b := get(s.x, s.y, s.z)
			below := get(s.x, s.y-1, s.z)
			if b != nil && b.name == "stairs" && below != nil && below.name == "stairs" &&
				below.props["half"] == b.props["half"] && below.props["facing"] == b.props["facing"] {
				set(s.x, s.y, s.z, "planks", nil)
			}
		}
	}

	// hwAt reports the hull half-width at a row, used by the decoration
	// passes (railings, gun ports, posts).
	hwAt := func(y, z int) int {
		if y < 0 || y > topY || z < 0 || z >= L {
			return -1
		}
		return hwArr[y][z]
	}
	has := hasHull
	// inHull seeds the connectivity cleanup.
	inHull := make(map[coord]bool)
	for z := 0; z < L; z++ {
		for y := 0; y <= topY; y++ {
			hw := hwArr[y][z]
			for x := -hw; x <= hw; x++ {
				inHull[coord{x, y, z}] = true
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
			// Vanilla facing = tall side; point it inboard so the cap
			// slopes down toward the post's outer end.
			facing := "north"
			if !fromBow {
				facing = "south"
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
