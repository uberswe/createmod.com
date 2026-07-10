package schematic

import (
	"bytes"
	"fmt"

	"github.com/Tnze/go-mc/nbt"
)

// Sponge Schematic Format (.schem), versions 2 and 3.
//
//	v2 root "Schematic" {
//	  Version: 2, DataVersion: int
//	  Width, Height, Length: short (unsigned)
//	  Palette: Compound{"minecraft:stone[facing=east]": int}, PaletteMax: int
//	  BlockData: byte array — unsigned varint palette indices, x-fastest
//	             order: index = x + Width*(z + Length*y)
//	  BlockEntities: List<Compound{Pos: int[3], Id: string, ...data}>
//	  Entities, Offset, Metadata
//	}
//
//	v3 root { Schematic: { Version: 3, DataVersion, Width/Height/Length,
//	  Blocks: {Palette, Data, BlockEntities: List<{Pos, Id, Data: {...}}>},
//	  Entities: List<{Pos, Id, Data}> , Offset, Metadata } }
//
// Reads accept v2 and v3; writes emit v3 (current spec) with sorted palettes
// for byte-stable output.

type spongeBlockContainer struct {
	Palette       map[string]int32 `nbt:"Palette"`
	Data          []byte           `nbt:"Data"`
	BlockEntities []nbt.RawMessage `nbt:"BlockEntities"`
}

type spongeV3Body struct {
	Version     int32                `nbt:"Version"`
	DataVersion int32                `nbt:"DataVersion"`
	Width       int16                `nbt:"Width"`
	Height      int16                `nbt:"Height"`
	Length      int16                `nbt:"Length"`
	Blocks      spongeBlockContainer `nbt:"Blocks"`
	Entities    []nbt.RawMessage     `nbt:"Entities"`
	Metadata    nbt.RawMessage       `nbt:"Metadata"`
}

type spongeV2Root struct {
	Version       int32            `nbt:"Version"`
	DataVersion   int32            `nbt:"DataVersion"`
	Width         int16            `nbt:"Width"`
	Height        int16            `nbt:"Height"`
	Length        int16            `nbt:"Length"`
	Palette       map[string]int32 `nbt:"Palette"`
	BlockData     []byte           `nbt:"BlockData"`
	BlockEntities []nbt.RawMessage `nbt:"BlockEntities"`
	TileEntities  []nbt.RawMessage `nbt:"TileEntities"` // v1 name, some tools still emit it
	Entities      []nbt.RawMessage `nbt:"Entities"`
	Metadata      nbt.RawMessage   `nbt:"Metadata"`
	// v3 nesting: when present, the payload lives here instead.
	Schematic *spongeV3Body `nbt:"Schematic"`
}

// ReadSponge parses a Sponge .schem (v2 or v3) into the normalized model.
func ReadSponge(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root spongeV2Root
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not a valid .schem: %w", err)
	}

	var (
		version, dataVersion int32
		w, h, l              int
		palette              map[string]int32
		blockData            []byte
		blockEntities        []nbt.RawMessage
		entities             []nbt.RawMessage
		sourceVersion        string
	)
	switch {
	case root.Schematic != nil && root.Schematic.Version >= 3:
		b := root.Schematic
		version, dataVersion = b.Version, b.DataVersion
		w, h, l = int(uint16(b.Width)), int(uint16(b.Height)), int(uint16(b.Length))
		palette, blockData, blockEntities = b.Blocks.Palette, b.Blocks.Data, b.Blocks.BlockEntities
		entities = b.Entities
		sourceVersion = "schem-v3"
	case root.Version >= 1 && root.Palette != nil:
		version, dataVersion = root.Version, root.DataVersion
		w, h, l = int(uint16(root.Width)), int(uint16(root.Height)), int(uint16(root.Length))
		palette, blockData = root.Palette, root.BlockData
		blockEntities = root.BlockEntities
		if blockEntities == nil {
			blockEntities = root.TileEntities
		}
		entities = root.Entities
		sourceVersion = fmt.Sprintf("schem-v%d", version)
	default:
		return nil, fmt.Errorf("schematic: not a recognizable .schem (no Version/Palette)")
	}

	if w <= 0 || h <= 0 || l <= 0 {
		return nil, fmt.Errorf("schematic: .schem has non-positive size %dx%dx%d", w, h, l)
	}
	if w > MaxDimension || h > MaxDimension || l > MaxDimension {
		return nil, fmt.Errorf("schematic: .schem size %dx%dx%d exceeds maximum dimension", w, h, l)
	}
	if v := w * h * l; v > MaxVolume {
		return nil, fmt.Errorf("schematic: .schem volume %d exceeds maximum %d", v, MaxVolume)
	}
	if len(palette) > MaxPaletteSize {
		return nil, fmt.Errorf("schematic: .schem palette size %d exceeds maximum", len(palette))
	}

	s := New(w, h, l)
	s.DataVersion = int(dataVersion)
	s.Meta.SourceFormat = sourceVersion

	// Palette: state-string → source index. Invert to source index → model index.
	maxIdx := int32(-1)
	for _, idx := range palette {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if maxIdx >= int32(MaxPaletteSize) {
		return nil, fmt.Errorf("schematic: .schem palette index %d exceeds maximum", maxIdx)
	}
	srcToModel := make([]int32, maxIdx+1)
	for i := range srcToModel {
		srcToModel[i] = -1
	}
	for stateStr, idx := range palette {
		if idx < 0 {
			return nil, fmt.Errorf("schematic: .schem palette has negative index for %q", stateStr)
		}
		st, err := ParseStateString(stateStr)
		if err != nil {
			return nil, err
		}
		srcToModel[idx] = s.PaletteIndex(st)
	}

	// BlockData: unsigned varints, same x-fastest layout as the model.
	vol := w * h * l
	pos := 0
	for i := 0; i < vol; i++ {
		v, n := uvarint(blockData[pos:])
		if n <= 0 {
			return nil, fmt.Errorf("schematic: .schem block data truncated at block %d", i)
		}
		pos += n
		if v > uint64(maxIdx) || srcToModel[v] < 0 {
			return nil, fmt.Errorf("schematic: .schem block %d references unknown palette index %d", i, v)
		}
		s.Blocks[i] = srcToModel[v]
	}
	if pos != len(blockData) {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes, ".schem block data had trailing bytes")
	}

	// Block entities → canonical form (structure-style payload with "id",
	// position separated out).
	if len(blockEntities) > MaxBlockEntities {
		return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
	}
	for i, raw := range blockEntities {
		fields, err := compoundFields(raw)
		if err != nil {
			return nil, fmt.Errorf("schematic: .schem block entity %d: %w", i, err)
		}
		posRaw, ok := fields["Pos"]
		if !ok {
			return nil, fmt.Errorf("schematic: .schem block entity %d missing Pos", i)
		}
		var bePos []int32
		if err := unmarshalRaw(posRaw, &bePos); err != nil || len(bePos) != 3 {
			return nil, fmt.Errorf("schematic: .schem block entity %d has bad Pos", i)
		}
		x, y, z := int(bePos[0]), int(bePos[1]), int(bePos[2])
		if x < 0 || x >= w || y < 0 || y >= h || z < 0 || z >= l {
			return nil, fmt.Errorf("schematic: .schem block entity %d outside bounds", i)
		}
		canonical := map[string]nbt.RawMessage{}
		if dataRaw, hasData := fields["Data"]; hasData && dataRaw.Type == nbt.TagCompound {
			// v3: payload nested under Data
			dataFields, err := compoundFields(dataRaw)
			if err != nil {
				return nil, fmt.Errorf("schematic: .schem block entity %d data: %w", i, err)
			}
			canonical = dataFields
		} else {
			// v2: payload inline
			for k, v := range fields {
				if k == "Pos" || k == "Id" {
					continue
				}
				canonical[k] = v
			}
		}
		if idRaw, ok := fields["Id"]; ok {
			if id, ok2 := stringFromRaw(idRaw); ok2 {
				canonical["id"] = rawString(id)
			}
		}
		s.BlockEntities = append(s.BlockEntities, BlockEntity{
			Pos: [3]int{x, y, z},
			Raw: compoundFromFields(canonical),
		})
	}

	if len(entities) > MaxEntities {
		return nil, fmt.Errorf("schematic: more than %d entities", MaxEntities)
	}
	s.Entities = entities
	if len(entities) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes, "entity coordinate spaces differ between formats; entities carried as-is")
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// spongeV3BlockEntityOut mirrors the v3 spec entry.
type spongeV3BlockEntityOut struct {
	Pos  intArray `nbt:"Pos"`
	ID   string   `nbt:"Id"`
	Data rawNBT   `nbt:"Data"`
}

type spongeV3BlocksOut struct {
	Palette       orderedPalette           `nbt:"Palette"`
	Data          []byte                   `nbt:"Data"`
	BlockEntities []spongeV3BlockEntityOut `nbt:"BlockEntities"`
}

type spongeV3Out struct {
	Version     int32             `nbt:"Version"`
	DataVersion int32             `nbt:"DataVersion"`
	Width       int16             `nbt:"Width"`
	Height      int16             `nbt:"Height"`
	Length      int16             `nbt:"Length"`
	Offset      intArray          `nbt:"Offset"`
	Blocks      spongeV3BlocksOut `nbt:"Blocks"`
	Entities    rawList          `nbt:"Entities"`
}

type spongeRootOut struct {
	Schematic spongeV3Out `nbt:"Schematic"`
}

// WriteSponge serializes the model as a gzip-compressed Sponge v3 .schem.
func WriteSponge(s *Schematic) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	w, h, l := s.Size[0], s.Size[1], s.Size[2]
	if w > 0xFFFF || h > 0xFFFF || l > 0xFFFF {
		return nil, fmt.Errorf("schematic: size %v exceeds .schem uint16 dimensions", s.Size)
	}

	// Sponge palettes are dense over used states; air must be present since
	// the data array covers every position.
	used := make([]bool, len(s.Palette))
	for _, idx := range s.Blocks {
		used[idx] = true
	}
	modelToOut := make([]int32, len(s.Palette))
	var entries []string
	for i, st := range s.Palette {
		if !used[i] {
			modelToOut[i] = -1
			continue
		}
		modelToOut[i] = int32(len(entries))
		entries = append(entries, st.Key())
	}

	var data bytes.Buffer
	for _, idx := range s.Blocks {
		putUvarint(&data, uint64(modelToOut[idx]))
	}

	var blockEntities []spongeV3BlockEntityOut
	for _, be := range s.BlockEntities {
		fields, err := compoundFields(be.Raw)
		if err != nil {
			return nil, fmt.Errorf("schematic: block entity at %v: %w", be.Pos, err)
		}
		id := ""
		if idRaw, ok := fields["id"]; ok {
			id, _ = stringFromRaw(idRaw)
			delete(fields, "id")
		}
		blockEntities = append(blockEntities, spongeV3BlockEntityOut{
			Pos:  intArray{int32(be.Pos[0]), int32(be.Pos[1]), int32(be.Pos[2])},
			ID:   id,
			Data: rawNBT(compoundFromFields(fields)),
		})
	}

	entities := rawList(s.Entities)

	root := spongeRootOut{Schematic: spongeV3Out{
		Version:     3,
		DataVersion: int32(s.DataVersion),
		Width:       int16(w),
		Height:      int16(h),
		Length:      int16(l),
		Offset:      intArray{0, 0, 0},
		Blocks: spongeV3BlocksOut{
			Palette:       orderedPalette(entries),
			Data:          data.Bytes(),
			BlockEntities: blockEntities,
		},
		Entities: entities,
	}}

	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, ""); err != nil {
		return nil, fmt.Errorf("schematic: encode .schem: %w", err)
	}
	return gzipBytes(buf.Bytes())
}
