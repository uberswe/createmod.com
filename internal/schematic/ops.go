package schematic

import (
	"fmt"
	"strconv"
	"strings"
)

// Editor operations: pure transforms over the model, applied by replaying an
// op log (undo/redo = moving a cursor through the log and replaying — no
// inverse operations to get wrong).

// Op is one editor operation. Type selects the transform; the other fields
// are interpreted per type. Regions are inclusive block coordinates.
type Op struct {
	Type string `json:"type"`
	Min  [3]int `json:"min,omitempty"`
	Max  [3]int `json:"max,omitempty"`
	// Axis for mirror ("x" or "z"); Steps for rotate (90° clockwise turns).
	Axis  string `json:"axis,omitempty"`
	Steps int    `json:"steps,omitempty"`
	// Block for fill (blockstate string, e.g. "minecraft:stone" or
	// "minecraft:oak_stairs[facing=east]").
	Block string `json:"block,omitempty"`
	// Replacements for replace: To == "" removes (becomes air).
	Replacements []OpReplacement `json:"replacements,omitempty"`
	// Expand growth per side: Min grows the low sides, Max the high sides.
	Grow struct {
		Low  [3]int `json:"low"`
		High [3]int `json:"high"`
	} `json:"grow,omitempty"`
}

type OpReplacement struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// MaxOpsPerSession bounds op-log replay cost.
const MaxOpsPerSession = 200

// ApplyOp returns a new model with the operation applied.
func ApplyOp(s *Schematic, op Op) (*Schematic, error) {
	switch op.Type {
	case "crop":
		return opCrop(s, op.Min, op.Max)
	case "expand":
		return opExpand(s, op.Grow.Low, op.Grow.High)
	case "rotate":
		return opRotate(s, op.Steps)
	case "mirror":
		return opMirror(s, op.Axis)
	case "fill":
		return opFill(s, op.Min, op.Max, op.Block, false)
	case "delete_region":
		return opFill(s, op.Min, op.Max, "minecraft:air", false)
	case "hollow":
		return opFill(s, op.Min, op.Max, "minecraft:air", true)
	case "replace":
		return opReplace(s, op.Replacements)
	default:
		return nil, fmt.Errorf("schematic: unknown operation %q", op.Type)
	}
}

// ApplyOps replays a log prefix.
func ApplyOps(s *Schematic, ops []Op) (*Schematic, error) {
	if len(ops) > MaxOpsPerSession {
		return nil, fmt.Errorf("schematic: op log exceeds %d operations", MaxOpsPerSession)
	}
	cur := s
	for i, op := range ops {
		next, err := ApplyOp(cur, op)
		if err != nil {
			return nil, fmt.Errorf("schematic: op %d (%s): %w", i, op.Type, err)
		}
		cur = next
	}
	return cur, nil
}

func clampRegion(s *Schematic, min, max [3]int) ([3]int, [3]int, error) {
	for a := 0; a < 3; a++ {
		if min[a] > max[a] {
			min[a], max[a] = max[a], min[a]
		}
		if min[a] < 0 {
			min[a] = 0
		}
		if max[a] >= s.Size[a] {
			max[a] = s.Size[a] - 1
		}
		if min[a] > max[a] {
			return min, max, fmt.Errorf("region outside the build")
		}
	}
	return min, max, nil
}

func opCrop(s *Schematic, min, max [3]int) (*Schematic, error) {
	min, max, err := clampRegion(s, min, max)
	if err != nil {
		return nil, err
	}
	out := New(max[0]-min[0]+1, max[1]-min[1]+1, max[2]-min[2]+1)
	out.DataVersion = s.DataVersion
	out.Meta = s.Meta
	out.Palette = clonePalette(s.Palette)
	out.Blocks = make([]int32, out.Volume())
	for y := min[1]; y <= max[1]; y++ {
		for z := min[2]; z <= max[2]; z++ {
			for x := min[0]; x <= max[0]; x++ {
				out.Blocks[out.Index(x-min[0], y-min[1], z-min[2])] = s.Blocks[s.Index(x, y, z)]
			}
		}
	}
	for _, be := range s.BlockEntities {
		if be.Pos[0] < min[0] || be.Pos[0] > max[0] || be.Pos[1] < min[1] || be.Pos[1] > max[1] || be.Pos[2] < min[2] || be.Pos[2] > max[2] {
			continue
		}
		out.BlockEntities = append(out.BlockEntities, BlockEntity{
			Pos: [3]int{be.Pos[0] - min[0], be.Pos[1] - min[1], be.Pos[2] - min[2]},
			Raw: be.Raw,
		})
	}
	return out, out.Validate()
}

func opExpand(s *Schematic, low, high [3]int) (*Schematic, error) {
	for a := 0; a < 3; a++ {
		if low[a] < 0 || high[a] < 0 || low[a] > 256 || high[a] > 256 {
			return nil, fmt.Errorf("growth must be between 0 and 256 per side")
		}
	}
	nx, ny, nz := s.Size[0]+low[0]+high[0], s.Size[1]+low[1]+high[1], s.Size[2]+low[2]+high[2]
	if nx > MaxDimension || ny > MaxDimension || nz > MaxDimension || nx*ny*nz > MaxVolume {
		return nil, fmt.Errorf("expanded size exceeds limits")
	}
	out := New(nx, ny, nz)
	out.DataVersion = s.DataVersion
	out.Meta = s.Meta
	out.Palette = clonePalette(s.Palette)
	out.Blocks = make([]int32, out.Volume())
	for y := 0; y < s.Size[1]; y++ {
		for z := 0; z < s.Size[2]; z++ {
			for x := 0; x < s.Size[0]; x++ {
				out.Blocks[out.Index(x+low[0], y+low[1], z+low[2])] = s.Blocks[s.Index(x, y, z)]
			}
		}
	}
	for _, be := range s.BlockEntities {
		out.BlockEntities = append(out.BlockEntities, BlockEntity{
			Pos: [3]int{be.Pos[0] + low[0], be.Pos[1] + low[1], be.Pos[2] + low[2]},
			Raw: be.Raw,
		})
	}
	return out, out.Validate()
}

var facingCW = map[string]string{"north": "east", "east": "south", "south": "west", "west": "north"}
var mirrorXFacing = map[string]string{"east": "west", "west": "east"}   // mirror across YZ plane (x flips)
var mirrorZFacing = map[string]string{"north": "south", "south": "north"} // mirror across XY plane (z flips)

// rotateProps rotates orientation-carrying blockstate properties 90° CW.
func rotateProps(p map[string]string) map[string]string {
	if len(p) == 0 {
		return p
	}
	out := make(map[string]string, len(p))
	for k, v := range p {
		switch k {
		case "facing":
			if nv, ok := facingCW[v]; ok {
				v = nv
			}
		case "axis":
			if v == "x" {
				v = "z"
			} else if v == "z" {
				v = "x"
			}
		case "rotation":
			if n, err := strconv.Atoi(v); err == nil {
				v = strconv.Itoa((n + 4) % 16)
			}
		}
		out[k] = v
	}
	return out
}

func mirrorProps(p map[string]string, axis string) map[string]string {
	if len(p) == 0 {
		return p
	}
	facingMap := mirrorXFacing
	if axis == "z" {
		facingMap = mirrorZFacing
	}
	out := make(map[string]string, len(p))
	for k, v := range p {
		switch k {
		case "facing":
			if nv, ok := facingMap[v]; ok {
				v = nv
			}
		case "shape":
			// stair corners swap handedness under mirroring
			if strings.HasSuffix(v, "_left") {
				v = strings.TrimSuffix(v, "_left") + "_right"
			} else if strings.HasSuffix(v, "_right") {
				v = strings.TrimSuffix(v, "_right") + "_left"
			}
		}
		out[k] = v
	}
	return out
}

func transformPalette(palette []BlockState, f func(map[string]string) map[string]string) []BlockState {
	out := make([]BlockState, len(palette))
	for i, st := range palette {
		out[i] = BlockState{Name: st.Name, Properties: f(st.Properties)}
	}
	return out
}

func clonePalette(palette []BlockState) []BlockState {
	return transformPalette(palette, func(p map[string]string) map[string]string { return p })
}

// opRotate rotates the model N 90° clockwise turns around Y.
func opRotate(s *Schematic, steps int) (*Schematic, error) {
	steps = ((steps % 4) + 4) % 4
	if steps == 0 {
		return s, nil
	}
	cur := s
	for i := 0; i < steps; i++ {
		// (x, z) -> (size_z-1-z, x)
		out := New(cur.Size[2], cur.Size[1], cur.Size[0])
		out.DataVersion = cur.DataVersion
		out.Meta = cur.Meta
		out.Palette = transformPalette(cur.Palette, rotateProps)
		out.Blocks = make([]int32, out.Volume())
		for y := 0; y < cur.Size[1]; y++ {
			for z := 0; z < cur.Size[2]; z++ {
				for x := 0; x < cur.Size[0]; x++ {
					out.Blocks[out.Index(cur.Size[2]-1-z, y, x)] = cur.Blocks[cur.Index(x, y, z)]
				}
			}
		}
		for _, be := range cur.BlockEntities {
			out.BlockEntities = append(out.BlockEntities, BlockEntity{
				Pos: [3]int{cur.Size[2] - 1 - be.Pos[2], be.Pos[1], be.Pos[0]},
				Raw: be.Raw,
			})
		}
		cur = out
	}
	return cur, cur.Validate()
}

func opMirror(s *Schematic, axis string) (*Schematic, error) {
	if axis != "x" && axis != "z" {
		return nil, fmt.Errorf("mirror axis must be x or z")
	}
	out := New(s.Size[0], s.Size[1], s.Size[2])
	out.DataVersion = s.DataVersion
	out.Meta = s.Meta
	out.Palette = transformPalette(s.Palette, func(p map[string]string) map[string]string {
		return mirrorProps(p, axis)
	})
	out.Blocks = make([]int32, out.Volume())
	for y := 0; y < s.Size[1]; y++ {
		for z := 0; z < s.Size[2]; z++ {
			for x := 0; x < s.Size[0]; x++ {
				nx, nz := x, z
				if axis == "x" {
					nx = s.Size[0] - 1 - x
				} else {
					nz = s.Size[2] - 1 - z
				}
				out.Blocks[out.Index(nx, y, nz)] = s.Blocks[s.Index(x, y, z)]
			}
		}
	}
	for _, be := range s.BlockEntities {
		pos := be.Pos
		if axis == "x" {
			pos[0] = s.Size[0] - 1 - pos[0]
		} else {
			pos[2] = s.Size[2] - 1 - pos[2]
		}
		out.BlockEntities = append(out.BlockEntities, BlockEntity{Pos: pos, Raw: be.Raw})
	}
	return out, out.Validate()
}

// opFill sets a region to a block; hollow keeps the region's outer shell.
func opFill(s *Schematic, min, max [3]int, block string, hollow bool) (*Schematic, error) {
	min, max, err := clampRegion(s, min, max)
	if err != nil {
		return nil, err
	}
	st, err := ParseStateString(block)
	if err != nil {
		return nil, err
	}
	if !isValidBlockID(st.Name) {
		return nil, fmt.Errorf("invalid block id %q", st.Name)
	}
	out := shallowCopy(s)
	idx := out.PaletteIndex(st)
	isAir := st.IsAir()
	for y := min[1]; y <= max[1]; y++ {
		for z := min[2]; z <= max[2]; z++ {
			for x := min[0]; x <= max[0]; x++ {
				if hollow {
					onShell := x == min[0] || x == max[0] || y == min[1] || y == max[1] || z == min[2] || z == max[2]
					if onShell {
						continue
					}
				}
				out.Blocks[out.Index(x, y, z)] = idx
			}
		}
	}
	// Drop block entities whose block was overwritten.
	kept := out.BlockEntities[:0]
	for _, be := range out.BlockEntities {
		inRegion := be.Pos[0] >= min[0] && be.Pos[0] <= max[0] && be.Pos[1] >= min[1] && be.Pos[1] <= max[1] && be.Pos[2] >= min[2] && be.Pos[2] <= max[2]
		if inRegion {
			if hollow {
				onShell := be.Pos[0] == min[0] || be.Pos[0] == max[0] || be.Pos[1] == min[1] || be.Pos[1] == max[1] || be.Pos[2] == min[2] || be.Pos[2] == max[2]
				if onShell {
					kept = append(kept, be)
				}
				continue
			}
			if !isAir {
				continue // overwritten by a plain block
			}
			continue
		}
		kept = append(kept, be)
	}
	out.BlockEntities = kept
	return out, out.Validate()
}

func opReplace(s *Schematic, replacements []OpReplacement) (*Schematic, error) {
	if len(replacements) == 0 || len(replacements) > 100 {
		return nil, fmt.Errorf("between 1 and 100 replacements required")
	}
	byName := map[string]string{}
	for _, r := range replacements {
		from := strings.TrimSpace(r.From)
		to := strings.TrimSpace(r.To)
		if to == "" {
			to = "minecraft:air"
		}
		if !isValidBlockID(from) || !isValidBlockID(to) {
			return nil, fmt.Errorf("invalid block id in replacement %q -> %q", from, to)
		}
		byName[from] = to
	}
	out := shallowCopy(s)
	out.Palette = clonePalette(s.Palette)
	dropped := map[int32]bool{}
	for i := range out.Palette {
		if to, ok := byName[out.Palette[i].Name]; ok {
			if to == "minecraft:air" {
				dropped[int32(i)] = true
			}
			// Keep properties when both blocks share a family suffix
			// (oak_stairs -> spruce_stairs keeps facing); drop otherwise.
			if !sameFamilySuffix(out.Palette[i].Name, to) {
				out.Palette[i].Properties = nil
			}
			out.Palette[i].Name = to
		}
	}
	if len(dropped) > 0 {
		airIdx := out.PaletteIndex(BlockState{Name: "minecraft:air"})
		blocks := make([]int32, len(out.Blocks))
		copy(blocks, out.Blocks)
		for i, idx := range blocks {
			if dropped[idx] {
				blocks[i] = airIdx
			}
		}
		out.Blocks = blocks
		kept := []BlockEntity{}
		for _, be := range out.BlockEntities {
			if !dropped[s.Blocks[s.Index(be.Pos[0], be.Pos[1], be.Pos[2])]] {
				kept = append(kept, be)
			}
		}
		out.BlockEntities = kept
	}
	return out, out.Validate()
}

func sameFamilySuffix(a, b string) bool {
	sa, sb := a, b
	if i := strings.LastIndexByte(a, '_'); i >= 0 {
		sa = a[i:]
	}
	if i := strings.LastIndexByte(b, '_'); i >= 0 {
		sb = b[i:]
	}
	return sa == sb
}

func isValidBlockID(id string) bool {
	if len(id) == 0 || len(id) > MaxBlockIDLength {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == ':' || c == '/' || c == '-' || c == '.') {
			return false
		}
	}
	return strings.Count(id, ":") <= 1
}

// shallowCopy shares the palette/BE slices' backing where safe and copies
// the block array (ops mutate blocks or palette, never both aliased).
func shallowCopy(s *Schematic) *Schematic {
	out := &Schematic{
		Size:          s.Size,
		Palette:       append([]BlockState{}, s.Palette...),
		Blocks:        make([]int32, len(s.Blocks)),
		BlockEntities: append([]BlockEntity{}, s.BlockEntities...),
		Entities:      s.Entities,
		DataVersion:   s.DataVersion,
		Meta:          s.Meta,
	}
	copy(out.Blocks, s.Blocks)
	return out
}
