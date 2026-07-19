package schematic

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"math"
	"sort"
)

// Structure-similarity fingerprints. Five independent components combine
// into a weighted overall percentage; the per-component scores are the
// user-visible breakdown. Compact (~1-2 KB) so a whole library's
// fingerprints fit in memory and brute-force comparison stays in the
// low milliseconds.

// FingerprintVersion is stored with computed fingerprints; bump when the
// grid size, family table or any encoding changes so the backfill
// recomputes everything.
const FingerprintVersion = 1

// shapeGridN is the voxel grid edge; 16³ coverage cells = 4 KB quantized.
const shapeGridN = 16

type Fingerprint struct {
	Version int `json:"version"`
	// Shape is the canonical-yaw occupancy bitset (x + N*(z + N*y) bit order).
	Shape []byte `json:"shape"`
	// Families is the block-family histogram (BlockFamilies order, raw counts).
	Families []float32 `json:"families"`
	// Dims is the bounding box sorted descending (rotation-invariant).
	Dims [3]int `json:"dims"`
	// FillDensity = blocks / volume.
	FillDensity float64 `json:"fillDensity"`
	BlockCount  int     `json:"blockCount"`
	// PaletteHashes are FNV-1a hashes of the used blockstate keys, sorted.
	PaletteHashes []uint64 `json:"paletteHashes"`
}

// ComponentScore is one part of the similarity breakdown.
type ComponentScore struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`  // 0..1
	Weight float64 `json:"weight"` // fraction of the overall
}

// Similarity is the comparison result: overall percentage plus breakdown.
type Similarity struct {
	Overall    float64          `json:"overall"` // 0..1
	Components []ComponentScore `json:"components"`
}

var componentWeights = []struct {
	name   string
	weight float64
}{
	{"shape", 0.30},
	{"materials", 0.30},
	{"function", 0.20},
	{"proportions", 0.10},
	{"palette", 0.10},
}

// ComputeFingerprint builds the fingerprint for a schematic model.
func ComputeFingerprint(s *Schematic) *Fingerprint {
	fp := &Fingerprint{Version: FingerprintVersion}

	// Family histogram + palette hashes over used states.
	used := make([]bool, len(s.Palette))
	perPalette := make([]int, len(s.Palette))
	blockCount := 0
	for _, idx := range s.Blocks {
		perPalette[idx]++
		used[idx] = true
	}
	fp.Families = make([]float32, len(BlockFamilies))
	var hashes []uint64
	for i, st := range s.Palette {
		if st.IsAir() {
			continue
		}
		n := perPalette[i]
		if n == 0 {
			continue
		}
		blockCount += n
		if fi, ok := familyIndex[BlockFamily(st.Name)]; ok {
			fp.Families[fi] += float32(n)
		}
		if used[i] {
			h := fnv.New64a()
			_, _ = h.Write([]byte(st.Key()))
			hashes = append(hashes, h.Sum64())
		}
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })
	fp.PaletteHashes = hashes
	fp.BlockCount = blockCount

	// Rotation-invariant dims: sorted descending.
	d := []int{s.Size[0], s.Size[1], s.Size[2]}
	sort.Sort(sort.Reverse(sort.IntSlice(d)))
	fp.Dims = [3]int{d[0], d[1], d[2]}
	if v := s.Volume(); v > 0 {
		fp.FillDensity = float64(blockCount) / float64(v)
	}

	// Shape grid at canonical yaw: build once, rotate the grid 4 ways, keep
	// the lexicographically smallest encoding.
	grid := buildShapeGrid(s)
	best := grid
	cur := grid
	for i := 0; i < 3; i++ {
		cur = rotateGridY(cur)
		if bytes.Compare(cur, best) < 0 {
			best = cur
		}
	}
	fp.Shape = best
	return fp
}

// axisSplit precomputes, for one source axis, how each coordinate's unit
// span distributes across grid cells (geometric overlap). Coverage is exact
// under reflection, which is what makes rotated/mirrored copies score 1.0.
type axisSpan struct {
	cell int
	w    float64
}

func axisSplits(size int) [][]axisSpan {
	out := make([][]axisSpan, size)
	scale := float64(shapeGridN) / float64(size)
	for i := 0; i < size; i++ {
		start := float64(i) * scale
		end := float64(i+1) * scale
		c0 := int(start)
		c1 := int(end)
		if c1 >= shapeGridN {
			c1 = shapeGridN - 1
		}
		var spans []axisSpan
		for c := c0; c <= c1; c++ {
			lo := math.Max(start, float64(c))
			hi := math.Min(end, float64(c+1))
			if hi > lo {
				spans = append(spans, axisSpan{cell: c, w: hi - lo})
			}
		}
		out[i] = spans
	}
	return out
}

// buildShapeGrid downsamples occupancy into a 16³ coverage grid, quantized
// to uint8 with the max cell normalized to 255 (scale-free).
func buildShapeGrid(s *Schematic) []byte {
	n := shapeGridN
	acc := make([]float64, n*n*n)
	sx, sy, sz := s.Size[0], s.Size[1], s.Size[2]
	xs, ys, zs := axisSplits(sx), axisSplits(sy), axisSplits(sz)
	for y := 0; y < sy; y++ {
		for z := 0; z < sz; z++ {
			for x := 0; x < sx; x++ {
				if s.Palette[s.Blocks[s.Index(x, y, z)]].IsAir() {
					continue
				}
				for _, wy := range ys[y] {
					for _, wz := range zs[z] {
						for _, wx := range xs[x] {
							acc[wx.cell+n*(wz.cell+n*wy.cell)] += wx.w * wy.w * wz.w
						}
					}
				}
			}
		}
	}
	maxV := 0.0
	for _, v := range acc {
		if v > maxV {
			maxV = v
		}
	}
	grid := make([]byte, n*n*n)
	if maxV > 0 {
		for i, v := range acc {
			grid[i] = byte(math.Round(255 * v / maxV))
		}
	}
	return grid
}

// rotateGridY rotates the coverage grid 90° around the vertical axis.
func rotateGridY(grid []byte) []byte {
	out := make([]byte, len(grid))
	n := shapeGridN
	for y := 0; y < n; y++ {
		for z := 0; z < n; z++ {
			for x := 0; x < n; x++ {
				// (x, z) -> (z, n-1-x)
				out[z+n*((n-1-x)+n*y)] = grid[x+n*(z+n*y)]
			}
		}
	}
	return out
}

// jaccardBits is the generalized (weighted) Jaccard over coverage grids:
// sum(min)/sum(max). Scale-free because grids are max-normalized.
func jaccardBits(a, b []byte) float64 {
	if len(a) != len(b) {
		return 0
	}
	var inter, union int
	for i := range a {
		if a[i] < b[i] {
			inter += int(a[i])
			union += int(b[i])
		} else {
			inter += int(b[i])
			union += int(a[i])
		}
	}
	if union == 0 {
		return 1 // two empty shapes are identical
	}
	return float64(inter) / float64(union)
}

func cosine32(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		if i >= len(b) {
			break
		}
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 && nb == 0 {
		return 1
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func jaccardHashes(a, b []uint64) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	i, j, inter := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			inter++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

// ratioCloseness compares positive scalars as min/max (1 = identical).
func ratioCloseness(a, b float64) float64 {
	if a <= 0 && b <= 0 {
		return 1
	}
	if a <= 0 || b <= 0 {
		return 0
	}
	if a > b {
		a, b = b, a
	}
	return a / b
}

// logDamp compresses raw family counts to log scale for comparison.
func logDamp(v []float32) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		if x > 0 {
			out[i] = float32(math.Log1p(float64(x)))
		}
	}
	return out
}

// functionalFraction is the share of a build's blocks that are functional
// Create components.
func functionalFraction(fn []float32, blockCount int) float64 {
	if blockCount <= 0 {
		return 0
	}
	var sum float64
	for _, v := range fn {
		sum += float64(v)
	}
	return sum / float64(blockCount)
}

// functionVector extracts the Create functional profile from the family
// histogram.
func functionVector(families []float32) []float32 {
	out := make([]float32, len(functionFamilies))
	for i, f := range functionFamilies {
		if fi, ok := familyIndex[f]; ok && fi < len(families) {
			out[i] = families[fi]
		}
	}
	return out
}

// Compare scores two fingerprints. Symmetric.
func Compare(a, b *Fingerprint) Similarity {
	scores := map[string]float64{}

	// Shape: best of 4 yaw rotations (canonicalization plus belt-and-braces).
	shapeScore := jaccardBits(a.Shape, b.Shape)
	rot := b.Shape
	for i := 0; i < 3; i++ {
		rot = rotateGridY(rot)
		if s := jaccardBits(a.Shape, rot); s > shapeScore {
			shapeScore = s
		}
	}
	scores["shape"] = shapeScore

	// Materials: log-damped counts, so one dominant family (usually the
	// terrain a build was scanned with) cannot hijack the cosine.
	scores["materials"] = cosine32(logDamp(a.Families), logDamp(b.Families))

	// Function: kind-of-machinery direction scaled by how much of each build
	// is machinery. Cosine alone is scale-invariant — a house with three
	// pipes would otherwise "function" like a giant boiler. Builds whose
	// functional share is below the noise floor count as plain non-machines:
	// two such builds agree perfectly, and a handful of decorative Create
	// blocks neither helps nor hurts.
	fa, fb := functionVector(a.Families), functionVector(b.Families)
	da := functionalFraction(fa, a.BlockCount)
	db := functionalFraction(fb, b.BlockCount)
	const functionalNoiseFloor = 0.02
	switch {
	case da < functionalNoiseFloor && db < functionalNoiseFloor:
		scores["function"] = 1
	case da < functionalNoiseFloor || db < functionalNoiseFloor:
		scores["function"] = ratioCloseness(math.Max(da, functionalNoiseFloor), math.Max(db, functionalNoiseFloor))
	default:
		scores["function"] = cosine32(fa, fb) * ratioCloseness(da, db)
	}

	// The function WEIGHT scales with how much machinery is actually
	// present. Two decoration-only builds trivially "agree" on function
	// (score 1), and at a fixed 20% weight that agreement drowned the
	// shape signal — a house query boosted every other house by a fifth
	// of the score for carrying no machinery at all. The weight now grows
	// with the more functional of the two builds (keeping Compare
	// symmetric): zero machinery contributes zero weight, a single
	// mechanical bearing in a house a couple of percent, and builds at or
	// above functionalSaturation machinery get the full nominal weight.
	// The freed weight is redistributed across the other components by
	// renormalizing below.
	const functionalSaturation = 0.15
	functionW := math.Min(1, math.Sqrt(math.Max(da, db)/functionalSaturation))

	// Proportions: aspect-ratio closeness (scale-free) plus density and
	// absolute-size closeness (scale-aware, so a 2x copy scores below 1).
	aAspect1 := ratioCloseness(float64(a.Dims[1])/float64(max1(a.Dims[0])), float64(b.Dims[1])/float64(max1(b.Dims[0])))
	aAspect2 := ratioCloseness(float64(a.Dims[2])/float64(max1(a.Dims[0])), float64(b.Dims[2])/float64(max1(b.Dims[0])))
	sizeScore := ratioCloseness(float64(a.Dims[0]), float64(b.Dims[0]))
	density := ratioCloseness(a.FillDensity, b.FillDensity)
	scores["proportions"] = (aAspect1 + aAspect2 + sizeScore + density) / 4

	scores["palette"] = jaccardHashes(a.PaletteHashes, b.PaletteHashes)

	totalW := 0.0
	for _, cw := range componentWeights {
		w := cw.weight
		if cw.name == "function" {
			w *= functionW
		}
		totalW += w
	}
	var sim Similarity
	for _, cw := range componentWeights {
		w := cw.weight
		if cw.name == "function" {
			w *= functionW
		}
		w /= totalW
		s := scores[cw.name]
		sim.Components = append(sim.Components, ComponentScore{Name: cw.name, Score: s, Weight: w})
		sim.Overall += s * w
	}
	return sim
}

func max1(v int) float64 {
	if v < 1 {
		return 1
	}
	return float64(v)
}

// EncodeFingerprint serializes for storage.
func EncodeFingerprint(fp *Fingerprint) ([]byte, error) { return json.Marshal(fp) }

// DecodeFingerprint deserializes a stored fingerprint.
func DecodeFingerprint(data []byte) (*Fingerprint, error) {
	var fp Fingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return nil, err
	}
	return &fp, nil
}
