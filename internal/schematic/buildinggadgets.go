package schematic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// Building Gadgets templates (.json / .txt paste strings), implemented from
// the MIT-licensed Direwolf20-MC/BuildingGadgets2 source.
//
// Building Gadgets 2 (modern):
//
//	{"name": "...", "statePosArrayList": "<SNBT>", "requiredItems": {...}}
//
// where the SNBT compound is
//
//	{ startpos: {X,Y,Z}, endpos: {X,Y,Z},
//	  blockstatemap: List<{Name, Properties}>,
//	  statelist: IntArray }
//
// with one statelist entry per position of the start..end box iterated in
// vanilla BlockPos.betweenClosed order: x fastest, then y, then z.
//
// Building Gadgets 1 (legacy paste strings, often shared as .txt):
//
//	{"header": {..., "bounding_box": {min_x..max_z}}, "body": "<base64>"}
//
// where body is gzipped NBT { pos: List<Long>, data: List<{state: {...}}> }
// and each long packs stateID(24b)<<40 | x(16b)<<24 | y(8b)<<16 | z(16b).
//
// Neither variant carries block entities or a DataVersion.

type bg2JSON struct {
	Name              string          `json:"name"`
	StatePosArrayList string          `json:"statePosArrayList"`
	RequiredItems     map[string]int  `json:"requiredItems,omitempty"`
	Header            json.RawMessage `json:"header,omitempty"` // BG1
	Body              string          `json:"body,omitempty"`   // BG1
}

type bgVec3 struct {
	X int32 `nbt:"X"`
	Y int32 `nbt:"Y"`
	Z int32 `nbt:"Z"`
}

type bg2StateMap struct {
	StartPos      bgVec3               `nbt:"startpos"`
	EndPos        bgVec3               `nbt:"endpos"`
	BlockStateMap []structPaletteEntry `nbt:"blockstatemap"`
	StateList     []int32              `nbt:"statelist"`
}

// looksLikeJSON reports whether data is a JSON document (BG templates are
// plain text, unlike every NBT-based format).
func looksLikeJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// snbtToFields parses an SNBT compound string into named raw fields.
func snbtDecode(snbt string, v interface{}) error {
	type wrap struct {
		V nbt.StringifiedMessage `nbt:"v"`
	}
	bin, err := nbt.Marshal(wrap{V: nbt.StringifiedMessage(snbt)})
	if err != nil {
		return fmt.Errorf("schematic: invalid SNBT: %w", err)
	}
	var out struct {
		V nbt.RawMessage `nbt:"v"`
	}
	if err := nbt.Unmarshal(bin, &out); err != nil {
		return err
	}
	return unmarshalRaw(out.V, v)
}

// bg2SNBT renders the state-map compound as SNBT by hand. The Tnze SNBT
// encoder cannot be used here: it suffixes TAG_Int_Array elements ("[I;1I]")
// which neither Minecraft's TagParser nor its own parser accept.
func bg2SNBT(sm bg2StateMapOut) string {
	var sb strings.Builder
	writePos := func(name string, v bgVec3) {
		fmt.Fprintf(&sb, "%s:{X:%d,Y:%d,Z:%d}", name, v.X, v.Y, v.Z)
	}
	sb.WriteByte('{')
	writePos("startpos", sm.StartPos)
	sb.WriteByte(',')
	writePos("endpos", sm.EndPos)
	sb.WriteString(",blockstatemap:[")
	for i, e := range sm.BlockStateMap {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{Name:`)
		sb.WriteString(quoteSNBT(e.Name))
		if len(e.Properties) > 0 {
			sb.WriteString(",Properties:{")
			keys := make([]string, 0, len(e.Properties))
			for k := range e.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for j, k := range keys {
				if j > 0 {
					sb.WriteByte(',')
				}
				sb.WriteString(k)
				sb.WriteByte(':')
				sb.WriteString(quoteSNBT(e.Properties[k]))
			}
			sb.WriteByte('}')
		}
		sb.WriteByte('}')
	}
	sb.WriteString("],statelist:[I;")
	for i, v := range sm.StateList {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%d", v)
	}
	sb.WriteString("]}")
	return sb.String()
}

// quoteSNBT double-quotes and escapes an SNBT string value.
func quoteSNBT(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"', '\\':
			sb.WriteByte('\\')
		}
		sb.WriteByte(s[i])
	}
	sb.WriteByte('"')
	return sb.String()
}

// ReadBuildingGadgets parses a Building Gadgets template (v2 JSON or v1
// paste string) into the normalized model.
func ReadBuildingGadgets(data []byte) (*Schematic, error) {
	if len(data) > MaxDecompressedSize {
		return nil, fmt.Errorf("schematic: template exceeds maximum size")
	}
	var doc bg2JSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("schematic: not a valid Building Gadgets template: %w", err)
	}
	switch {
	case doc.StatePosArrayList != "":
		return readBG2(doc)
	case doc.Body != "":
		return readBG1(doc)
	default:
		return nil, fmt.Errorf("schematic: JSON is not a Building Gadgets template (no statePosArrayList or body)")
	}
}

func readBG2(doc bg2JSON) (*Schematic, error) {
	var sm bg2StateMap
	if err := snbtDecode(doc.StatePosArrayList, &sm); err != nil {
		return nil, fmt.Errorf("schematic: Building Gadgets state list: %w", err)
	}
	if len(sm.BlockStateMap) == 0 {
		return nil, fmt.Errorf("schematic: Building Gadgets template has no blockstate map")
	}
	minX, maxX := order32(sm.StartPos.X, sm.EndPos.X)
	minY, maxY := order32(sm.StartPos.Y, sm.EndPos.Y)
	minZ, maxZ := order32(sm.StartPos.Z, sm.EndPos.Z)
	sx, sy, sz := int(maxX-minX)+1, int(maxY-minY)+1, int(maxZ-minZ)+1
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: template size exceeds maximum dimension")
	}
	vol := sx * sy * sz
	if vol > MaxVolume {
		return nil, fmt.Errorf("schematic: template volume exceeds maximum")
	}
	if len(sm.StateList) < vol {
		return nil, fmt.Errorf("schematic: template state list truncated (%d entries, need %d)", len(sm.StateList), vol)
	}

	s := New(sx, sy, sz)
	s.Meta.SourceFormat = "bg"
	s.Meta.Name = doc.Name
	s.Meta.LossyNotes = append(s.Meta.LossyNotes,
		"Building Gadgets templates carry no Minecraft data version or block entities")

	srcToModel := make([]int32, len(sm.BlockStateMap))
	for i, e := range sm.BlockStateMap {
		if len(e.Name) == 0 || len(e.Name) > MaxBlockIDLength {
			return nil, fmt.Errorf("schematic: template blockstate %d invalid", i)
		}
		srcToModel[i] = s.PaletteIndex(BlockState{Name: e.Name, Properties: e.Properties})
	}

	// statelist is iterated in vanilla betweenClosed order: x, then y, then z.
	i := 0
	for z := 0; z < sz; z++ {
		for y := 0; y < sy; y++ {
			for x := 0; x < sx; x++ {
				idx := sm.StateList[i]
				i++
				if idx < 0 || int(idx) >= len(srcToModel) {
					continue
				}
				s.Blocks[s.Index(x, y, z)] = srcToModel[idx]
			}
		}
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type bg1Body struct {
	Pos  []int64          `nbt:"pos"`
	Data []nbt.RawMessage `nbt:"data"`
}

type bg1Header struct {
	BoundingBox struct {
		MinX int `json:"min_x"`
		MinY int `json:"min_y"`
		MinZ int `json:"min_z"`
		MaxX int `json:"max_x"`
		MaxY int `json:"max_y"`
		MaxZ int `json:"max_z"`
	} `json:"bounding_box"`
}

func readBG1(doc bg2JSON) (*Schematic, error) {
	var header bg1Header
	if len(doc.Header) > 0 {
		if err := json.Unmarshal(doc.Header, &header); err != nil {
			return nil, fmt.Errorf("schematic: Building Gadgets v1 header: %w", err)
		}
	}
	raw, err := base64.StdEncoding.DecodeString(doc.Body)
	if err != nil {
		return nil, fmt.Errorf("schematic: Building Gadgets v1 body: %w", err)
	}
	payload, err := decompress(raw)
	if err != nil {
		return nil, err
	}
	var body bg1Body
	if err := nbt.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("schematic: Building Gadgets v1 payload: %w", err)
	}
	if len(body.Pos) == 0 {
		return nil, fmt.Errorf("schematic: Building Gadgets v1 template is empty")
	}
	if len(body.Pos) > MaxVolume {
		return nil, fmt.Errorf("schematic: template exceeds maximum volume")
	}

	type entry struct {
		x, y, z int
		state   int
	}
	entries := make([]entry, 0, len(body.Pos))
	minX, minY, minZ := 1<<30, 1<<30, 1<<30
	maxX, maxY, maxZ := -(1 << 30), -(1 << 30), -(1 << 30)
	for _, l := range body.Pos {
		e := entry{
			x:     int((l >> 24) & 0xFFFF),
			y:     int((l >> 16) & 0xFF),
			z:     int(l & 0xFFFF),
			state: int((l >> 40) & 0xFFFFFF),
		}
		entries = append(entries, e)
		if e.x < minX {
			minX = e.x
		}
		if e.y < minY {
			minY = e.y
		}
		if e.z < minZ {
			minZ = e.z
		}
		if e.x > maxX {
			maxX = e.x
		}
		if e.y > maxY {
			maxY = e.y
		}
		if e.z > maxZ {
			maxZ = e.z
		}
	}
	sx, sy, sz := maxX-minX+1, maxY-minY+1, maxZ-minZ+1
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: template size exceeds maximum dimension")
	}
	if v := sx * sy * sz; v > MaxVolume {
		return nil, fmt.Errorf("schematic: template volume exceeds maximum")
	}

	s := New(sx, sy, sz)
	s.Meta.SourceFormat = "bg"
	s.Meta.LossyNotes = append(s.Meta.LossyNotes,
		"Building Gadgets templates carry no Minecraft data version or block entities")

	// data: List<{state: {...blockstate...}}>
	states := make([]int32, len(body.Data))
	for i, raw := range body.Data {
		fields, err := compoundFields(raw)
		if err != nil {
			return nil, fmt.Errorf("schematic: Building Gadgets v1 state %d: %w", i, err)
		}
		var st structPaletteEntry
		if sRaw, ok := fields["state"]; ok {
			if err := unmarshalRaw(sRaw, &st); err != nil {
				return nil, fmt.Errorf("schematic: Building Gadgets v1 state %d: %w", i, err)
			}
		}
		if st.Name == "" {
			states[i] = 0
			continue
		}
		states[i] = s.PaletteIndex(BlockState{Name: st.Name, Properties: st.Properties})
	}
	for _, e := range entries {
		if e.state < 0 || e.state >= len(states) {
			continue
		}
		s.Blocks[s.Index(e.x-minX, e.y-minY, e.z-minZ)] = states[e.state]
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type bg2StateMapOut struct {
	StartPos      bgVec3                  `nbt:"startpos"`
	EndPos        bgVec3                  `nbt:"endpos"`
	BlockStateMap []structPaletteEntryOut `nbt:"blockstatemap"`
	StateList     []int32                 `nbt:"statelist"`
}

// WriteBuildingGadgets serializes the model as a Building Gadgets 2 template
// (.json). Block entities cannot be represented and are dropped with a
// warning; callers should surface the returned warnings.
func WriteBuildingGadgets(s *Schematic) ([]byte, []Warning, error) {
	if err := s.Validate(); err != nil {
		return nil, nil, err
	}
	var warnings []Warning
	if len(s.BlockEntities) > 0 {
		warnings = append(warnings, Warning{Message: fmt.Sprintf(
			"Building Gadgets templates cannot carry block entity data; %d block entities dropped (chest contents, Create kinetics, signs)", len(s.BlockEntities))})
	}
	if len(s.Entities) > 0 {
		warnings = append(warnings, Warning{Message: fmt.Sprintf("%d entities dropped", len(s.Entities))})
	}

	used := make([]bool, len(s.Palette))
	for _, idx := range s.Blocks {
		used[idx] = true
	}
	modelToOut := make([]int32, len(s.Palette))
	var outMap []structPaletteEntryOut
	for i, st := range s.Palette {
		if !used[i] {
			modelToOut[i] = -1
			continue
		}
		modelToOut[i] = int32(len(outMap))
		outMap = append(outMap, structPaletteEntryOut{Name: st.Name, Properties: st.Properties})
	}

	sx, sy, sz := s.Size[0], s.Size[1], s.Size[2]
	stateList := make([]int32, 0, s.Volume())
	counts := map[string]int{}
	for z := 0; z < sz; z++ {
		for y := 0; y < sy; y++ {
			for x := 0; x < sx; x++ {
				modelIdx := s.Blocks[s.Index(x, y, z)]
				stateList = append(stateList, modelToOut[modelIdx])
				st := s.Palette[modelIdx]
				if !st.IsAir() {
					counts[st.Name]++
				}
			}
		}
	}

	snbt := bg2SNBT(bg2StateMapOut{
		StartPos:      bgVec3{0, 0, 0},
		EndPos:        bgVec3{int32(sx - 1), int32(sy - 1), int32(sz - 1)},
		BlockStateMap: outMap,
		StateList:     stateList,
	})

	name := s.Meta.Name
	if name == "" {
		name = "schematic"
	}
	doc := bg2JSON{
		Name:              name,
		StatePosArrayList: snbt,
		RequiredItems:     counts,
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
}

func order32(a, b int32) (int32, int32) {
	if a <= b {
		return a, b
	}
	return b, a
}

func sortStrings(s []string) { sort.Strings(s) }
