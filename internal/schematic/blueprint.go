package schematic

import (
	"bytes"
	"fmt"

	"github.com/Tnze/go-mc/nbt"
)

// Structurize / MineColonies blueprint (.blueprint), format version 1
// (implemented from the published format in ldtteam/structurize's
// BlueprintUtil — clean-room reader/writer, no code or data reused):
//
//	root {
//	  version: Byte (1)
//	  size_x, size_y, size_z: Short
//	  palette: List<Compound>{Name, Properties} (vanilla blockstate codec)
//	  blocks: IntArray — palette indices as packed uint16 pairs, first index
//	          in the high 16 bits; positions x-fastest YZX order (the
//	          model's native layout); odd totals padded with zero
//	  tile_entities: List<Compound> (vanilla, x/y/z inline)
//	  entities: List<Compound>
//	  required_mods: List<String>
//	  name: String, architects: List<String>
//	  mcversion: Int — a Minecraft DataVersion
//	}

type blueprintRootIn struct {
	Version      int8                 `nbt:"version"`
	SizeX        int16                `nbt:"size_x"`
	SizeY        int16                `nbt:"size_y"`
	SizeZ        int16                `nbt:"size_z"`
	Palette      []structPaletteEntry `nbt:"palette"`
	Blocks       []int32              `nbt:"blocks"`
	TileEntities []nbt.RawMessage     `nbt:"tile_entities"`
	Entities     []nbt.RawMessage     `nbt:"entities"`
	RequiredMods []string             `nbt:"required_mods"`
	Name         string               `nbt:"name"`
	MCVersion    int32                `nbt:"mcversion"`
}

// ReadBlueprint parses a Structurize .blueprint into the normalized model.
func ReadBlueprint(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root blueprintRootIn
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not a valid .blueprint: %w", err)
	}
	if root.Version != 1 {
		return nil, fmt.Errorf("schematic: unsupported blueprint version %d", root.Version)
	}
	sx, sy, sz := int(root.SizeX), int(root.SizeY), int(root.SizeZ)
	if sx <= 0 || sy <= 0 || sz <= 0 {
		return nil, fmt.Errorf("schematic: .blueprint has non-positive size %dx%dx%d", sx, sy, sz)
	}
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: .blueprint size exceeds maximum dimension")
	}
	vol := sx * sy * sz
	if vol > MaxVolume {
		return nil, fmt.Errorf("schematic: .blueprint volume %d exceeds maximum", vol)
	}
	if len(root.Palette) == 0 || len(root.Palette) > MaxPaletteSize {
		return nil, fmt.Errorf("schematic: .blueprint palette size %d invalid", len(root.Palette))
	}
	if need := (vol + 1) / 2; len(root.Blocks) < need {
		return nil, fmt.Errorf("schematic: .blueprint block data truncated (%d ints, need %d)", len(root.Blocks), need)
	}

	s := New(sx, sy, sz)
	s.DataVersion = int(root.MCVersion)
	s.Meta.SourceFormat = "blueprint"
	s.Meta.Name = root.Name

	srcToModel := make([]int32, len(root.Palette))
	for i, e := range root.Palette {
		if len(e.Name) == 0 || len(e.Name) > MaxBlockIDLength {
			return nil, fmt.Errorf("schematic: .blueprint palette entry %d invalid", i)
		}
		// Structurize uses a substitution block as its "air" for structure
		// voids; carry it as-is (it converts by name like any other block).
		srcToModel[i] = s.PaletteIndex(BlockState{Name: e.Name, Properties: e.Properties})
	}

	for i := 0; i < vol; i++ {
		packed := uint32(root.Blocks[i/2])
		var idx uint16
		if i%2 == 0 {
			idx = uint16(packed >> 16)
		} else {
			idx = uint16(packed & 0xFFFF)
		}
		if int(idx) >= len(srcToModel) {
			return nil, fmt.Errorf("schematic: .blueprint block %d references palette index %d out of range", i, idx)
		}
		s.Blocks[i] = srcToModel[idx]
	}

	if len(root.TileEntities) > MaxBlockEntities {
		return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
	}
	for _, teRaw := range root.TileEntities {
		fields, err := compoundFields(teRaw)
		if err != nil {
			return nil, fmt.Errorf("schematic: .blueprint tile entity: %w", err)
		}
		px, okx := intFromRaw(fields["x"])
		py, oky := intFromRaw(fields["y"])
		pz, okz := intFromRaw(fields["z"])
		if !okx || !oky || !okz {
			continue
		}
		if int(px) < 0 || int(px) >= sx || int(py) < 0 || int(py) >= sy || int(pz) < 0 || int(pz) >= sz {
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
	if len(root.Entities) > MaxEntities {
		return nil, fmt.Errorf("schematic: more than %d entities", MaxEntities)
	}
	s.Entities = root.Entities
	if len(root.Entities) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes, "entity coordinate spaces differ between formats; entities carried as-is")
	}
	if len(root.RequiredMods) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf(".blueprint declares required mods: %v", root.RequiredMods))
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type blueprintRootOut struct {
	Version      int8                    `nbt:"version"`
	SizeX        int16                   `nbt:"size_x"`
	SizeY        int16                   `nbt:"size_y"`
	SizeZ        int16                   `nbt:"size_z"`
	Palette      []structPaletteEntryOut `nbt:"palette"`
	Blocks       []int32                 `nbt:"blocks"`
	TileEntities rawList                 `nbt:"tile_entities"`
	Entities     rawList                 `nbt:"entities"`
	RequiredMods []string                `nbt:"required_mods"`
	Name         string                  `nbt:"name"`
	MCVersion    int32                   `nbt:"mcversion"`
}

// WriteBlueprint serializes the model as a gzip-compressed Structurize
// .blueprint (format version 1).
func WriteBlueprint(s *Schematic) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if s.Size[0] > 0x7FFF || s.Size[1] > 0x7FFF || s.Size[2] > 0x7FFF {
		return nil, fmt.Errorf("schematic: size %v exceeds .blueprint int16 dimensions", s.Size)
	}
	if len(s.Palette) > 0xFFFF {
		return nil, fmt.Errorf("schematic: palette exceeds .blueprint uint16 indices")
	}

	outPalette := make([]structPaletteEntryOut, len(s.Palette))
	mods := map[string]bool{}
	for i, st := range s.Palette {
		outPalette[i] = structPaletteEntryOut{Name: st.Name, Properties: st.Properties}
		if ns, _, ok := cutNamespace(st.Name); ok && ns != "minecraft" {
			mods[ns] = true
		}
	}
	requiredMods := make([]string, 0, len(mods))
	for ns := range mods {
		requiredMods = append(requiredMods, ns)
	}
	sortStrings(requiredMods)

	vol := s.Volume()
	packed := make([]int32, (vol+1)/2)
	for i := 0; i < vol; i++ {
		idx := uint32(uint16(s.Blocks[i]))
		if i%2 == 0 {
			packed[i/2] |= int32(idx << 16)
		} else {
			packed[i/2] |= int32(idx)
		}
	}

	tiles := make(rawList, 0, len(s.BlockEntities))
	for _, be := range s.BlockEntities {
		fields, err := compoundFields(be.Raw)
		if err != nil {
			return nil, fmt.Errorf("schematic: block entity at %v: %w", be.Pos, err)
		}
		fields["x"] = rawInt(int32(be.Pos[0]))
		fields["y"] = rawInt(int32(be.Pos[1]))
		fields["z"] = rawInt(int32(be.Pos[2]))
		tiles = append(tiles, compoundFromFields(fields))
	}

	name := s.Meta.Name
	if name == "" {
		name = "schematic"
	}
	root := blueprintRootOut{
		Version:      1,
		SizeX:        int16(s.Size[0]),
		SizeY:        int16(s.Size[1]),
		SizeZ:        int16(s.Size[2]),
		Palette:      outPalette,
		Blocks:       packed,
		TileEntities: tiles,
		Entities:     rawList(s.Entities),
		RequiredMods: requiredMods,
		Name:         name,
		MCVersion:    int32(s.DataVersion),
	}
	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, ""); err != nil {
		return nil, fmt.Errorf("schematic: encode .blueprint: %w", err)
	}
	return gzipBytes(buf.Bytes())
}

// cutNamespace splits "ns:path" block ids.
func cutNamespace(id string) (ns, path string, ok bool) {
	for i := 0; i < len(id); i++ {
		if id[i] == ':' {
			return id[:i], id[i+1:], true
		}
	}
	return "", id, false
}
