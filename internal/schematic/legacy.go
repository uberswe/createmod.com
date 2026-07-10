package schematic

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Tnze/go-mc/nbt"
)

// Legacy MCEdit/WorldEdit .schematic (pre-1.13):
//
//	root "Schematic" {
//	  Width, Height, Length: short
//	  Materials: "Alpha"
//	  Blocks: byte array (numeric block ids, x-fastest YZX order)
//	  Data: byte array (4-bit metadata per block)
//	  AddBlocks: optional nibble array extending ids above 255
//	  TileEntities: List<Compound> (x/y/z inline, pre-flattening ids)
//	  Entities: List
//	}
//
// Reading maps id:meta pairs through the 1.13 flattening table to modern
// blockstates. Writing is the reverse and is inherently lossy: modern blocks
// with no pre-1.13 equivalent become air, counted in a warning.

//go:embed legacydata/legacy_blocks.json
var legacyBlocksJSON []byte

// legacyDataVersion is 1.13.2 — the flattening reference the mapping table
// targets. Later tools upgrade from it via DataFixerUpper.
const legacyDataVersion = 1631

var legacyOnce sync.Once
var legacyToModern map[string]BlockState // "id:meta" -> state
var modernToLegacy map[string][2]byte    // state key -> {id, meta}
var modernNameToLegacy map[string][2]byte

func legacyTables() (map[string]BlockState, map[string][2]byte, map[string][2]byte) {
	legacyOnce.Do(func() {
		var raw map[string]string
		if err := json.Unmarshal(legacyBlocksJSON, &raw); err != nil {
			panic("schematic: embedded legacy_blocks.json is invalid: " + err.Error())
		}
		legacyToModern = make(map[string]BlockState, len(raw))
		modernToLegacy = make(map[string][2]byte, len(raw))
		modernNameToLegacy = make(map[string][2]byte, len(raw))
		// Deterministic iteration so first-wins reverse entries are stable.
		keys := make([]string, 0, len(raw))
		for k := range raw {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			st, err := ParseStateString(raw[k])
			if err != nil {
				continue
			}
			legacyToModern[k] = st
			var id, meta int
			if _, err := fmt.Sscanf(k, "%d:%d", &id, &meta); err != nil || id > 255 || meta > 15 {
				continue
			}
			pair := [2]byte{byte(id), byte(meta)}
			if _, ok := modernToLegacy[st.Key()]; !ok {
				modernToLegacy[st.Key()] = pair
			}
			// Name-only fallback prefers the meta:0 variant.
			if _, ok := modernNameToLegacy[st.Name]; !ok || meta == 0 {
				if cur, ok := modernNameToLegacy[st.Name]; !ok || (meta == 0 && cur[1] != 0) {
					modernNameToLegacy[st.Name] = pair
				}
			}
		}
	})
	return legacyToModern, modernToLegacy, modernNameToLegacy
}

type legacyRootIn struct {
	Width        int16            `nbt:"Width"`
	Height       int16            `nbt:"Height"`
	Length       int16            `nbt:"Length"`
	Materials    string           `nbt:"Materials"`
	Blocks       []byte           `nbt:"Blocks"`
	Data         []byte           `nbt:"Data"`
	AddBlocks    []byte           `nbt:"AddBlocks"`
	TileEntities []nbt.RawMessage `nbt:"TileEntities"`
	Entities     []nbt.RawMessage `nbt:"Entities"`
}

// ReadLegacy parses a pre-1.13 .schematic, flattening id:meta pairs to
// modern blockstates.
func ReadLegacy(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root legacyRootIn
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not a valid legacy .schematic: %w", err)
	}
	w, h, l := int(uint16(root.Width)), int(uint16(root.Height)), int(uint16(root.Length))
	if w <= 0 || h <= 0 || l <= 0 {
		return nil, fmt.Errorf("schematic: .schematic has non-positive size %dx%dx%d", w, h, l)
	}
	if w > MaxDimension || h > MaxDimension || l > MaxDimension {
		return nil, fmt.Errorf("schematic: .schematic size exceeds maximum dimension")
	}
	vol := w * h * l
	if vol > MaxVolume {
		return nil, fmt.Errorf("schematic: .schematic volume %d exceeds maximum", vol)
	}
	if len(root.Blocks) < vol || len(root.Data) < vol {
		return nil, fmt.Errorf("schematic: .schematic block/data arrays shorter than volume")
	}

	toModern, _, _ := legacyTables()

	s := New(w, h, l)
	s.DataVersion = legacyDataVersion
	s.Meta.SourceFormat = "schematic"
	s.Meta.LossyNotes = append(s.Meta.LossyNotes,
		"legacy .schematic blocks flattened to modern 1.13+ blockstates")
	if root.Materials != "" && root.Materials != "Alpha" {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf(".schematic Materials=%q; only Alpha ids are mapped reliably", root.Materials))
	}

	unknown := map[string]int{}
	for i := 0; i < vol; i++ {
		id := int(root.Blocks[i]) & 0xff
		if len(root.AddBlocks) > i>>1 {
			// AddBlocks packs two 4-bit id extensions per byte.
			if i&1 == 0 {
				id |= int(root.AddBlocks[i>>1]&0x0f) << 8
			} else {
				id |= int(root.AddBlocks[i>>1]&0xf0) << 4
			}
		}
		meta := int(root.Data[i]) & 0x0f
		if id == 0 {
			continue
		}
		st, ok := toModern[fmt.Sprintf("%d:%d", id, meta)]
		if !ok {
			st, ok = toModern[fmt.Sprintf("%d:0", id)]
		}
		if !ok {
			unknown[fmt.Sprintf("%d:%d", id, meta)]++
			continue
		}
		s.Blocks[i] = s.PaletteIndex(st)
	}
	if len(unknown) > 0 {
		total := 0
		ids := make([]string, 0, len(unknown))
		for k, c := range unknown {
			total += c
			ids = append(ids, k)
		}
		sort.Strings(ids)
		if len(ids) > 8 {
			ids = ids[:8]
		}
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf("%d blocks with unmapped legacy ids (%s) became air", total, strings.Join(ids, ", ")))
	}

	if len(root.TileEntities) > MaxBlockEntities {
		return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
	}
	for _, teRaw := range root.TileEntities {
		fields, err := compoundFields(teRaw)
		if err != nil {
			return nil, fmt.Errorf("schematic: .schematic tile entity: %w", err)
		}
		px, okx := intFromRaw(fields["x"])
		py, oky := intFromRaw(fields["y"])
		pz, okz := intFromRaw(fields["z"])
		if !okx || !oky || !okz {
			continue
		}
		if int(px) < 0 || int(px) >= w || int(py) < 0 || int(py) >= h || int(pz) < 0 || int(pz) >= l {
			continue
		}
		delete(fields, "x")
		delete(fields, "y")
		delete(fields, "z")
		s.BlockEntities = append(s.BlockEntities, BlockEntity{
			Pos: [3]int{int(px), int(py), int(pz)},
			Raw: compoundFromFields(fields),
		})
	}
	if len(s.BlockEntities) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			"tile entity ids/payloads are pre-1.13 and may need updating in-game")
	}
	if len(root.Entities) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf("%d legacy entities dropped", len(root.Entities)))
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type legacyRootOut struct {
	Width        int16   `nbt:"Width"`
	Height       int16   `nbt:"Height"`
	Length       int16   `nbt:"Length"`
	Materials    string  `nbt:"Materials"`
	Blocks       []byte  `nbt:"Blocks"`
	Data         []byte  `nbt:"Data"`
	TileEntities rawList `nbt:"TileEntities"`
	Entities     rawList `nbt:"Entities"`
}

// WriteLegacy serializes the model as a pre-1.13 .schematic. Inherently
// lossy: blockstate properties collapse to 4-bit metadata via the reverse
// flattening table, and modern blocks without a legacy equivalent become
// air. Callers should surface WarningsForLegacy alongside the bytes.
func WriteLegacy(s *Schematic) ([]byte, []Warning, error) {
	if err := s.Validate(); err != nil {
		return nil, nil, err
	}
	w, h, l := s.Size[0], s.Size[1], s.Size[2]
	if w > 0xFFFF || h > 0xFFFF || l > 0xFFFF {
		return nil, nil, fmt.Errorf("schematic: size %v exceeds .schematic uint16 dimensions", s.Size)
	}
	_, toLegacy, nameToLegacy := legacyTables()

	var warnings []Warning
	warnings = append(warnings, Warning{Message: "legacy .schematic export collapses modern blockstates to pre-1.13 id:meta pairs"})

	blocks := make([]byte, s.Volume())
	metas := make([]byte, s.Volume())
	// Per-palette resolution, resolved once.
	type legacyRes struct {
		pair  [2]byte
		found bool
	}
	resolved := make([]legacyRes, len(s.Palette))
	for i, st := range s.Palette {
		if st.IsAir() {
			resolved[i] = legacyRes{found: true} // 0:0
			continue
		}
		if pair, ok := toLegacy[st.Key()]; ok {
			resolved[i] = legacyRes{pair: pair, found: true}
			continue
		}
		if pair, ok := nameToLegacy[st.Name]; ok {
			resolved[i] = legacyRes{pair: pair, found: true}
			continue
		}
		resolved[i] = legacyRes{found: false}
	}
	dropped := map[string]int{}
	for i, idx := range s.Blocks {
		r := resolved[idx]
		if !r.found {
			dropped[s.Palette[idx].Name]++
			continue
		}
		blocks[i] = r.pair[0]
		metas[i] = r.pair[1]
	}
	if len(dropped) > 0 {
		total := 0
		names := make([]string, 0, len(dropped))
		for n, c := range dropped {
			total += c
			names = append(names, n)
		}
		sort.Strings(names)
		if len(names) > 8 {
			names = names[:8]
		}
		warnings = append(warnings, Warning{Message: fmt.Sprintf(
			"%d blocks have no pre-1.13 equivalent and became air (%s)", total, strings.Join(names, ", "))})
	}

	tiles := make(rawList, 0, len(s.BlockEntities))
	for _, be := range s.BlockEntities {
		fields, err := compoundFields(be.Raw)
		if err != nil {
			return nil, nil, fmt.Errorf("schematic: block entity at %v: %w", be.Pos, err)
		}
		fields["x"] = rawInt(int32(be.Pos[0]))
		fields["y"] = rawInt(int32(be.Pos[1]))
		fields["z"] = rawInt(int32(be.Pos[2]))
		tiles = append(tiles, compoundFromFields(fields))
	}
	if len(tiles) > 0 {
		warnings = append(warnings, Warning{Message: "block entity payloads use modern ids; pre-1.13 tools may not recognize them"})
	}

	root := legacyRootOut{
		Width: int16(w), Height: int16(h), Length: int16(l),
		Materials: "Alpha",
		Blocks:    blocks,
		Data:      metas,
		TileEntities: tiles,
		Entities:     rawList{},
	}
	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, "Schematic"); err != nil {
		return nil, nil, fmt.Errorf("schematic: encode .schematic: %w", err)
	}
	out, err := gzipBytes(buf.Bytes())
	if err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
}
