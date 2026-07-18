package schematic

import (
	"bytes"
	"fmt"
	"math/bits"

	"github.com/Tnze/go-mc/nbt"
)

// Litematica format (.litematic), versions 5/6.
//
//	root {
//	  MinecraftDataVersion, Version: int
//	  Metadata: { Name, Author, EnclosingSize{x,y,z}, RegionCount, ... }
//	  Regions: { "<name>": {
//	    Position{x,y,z}, Size{x,y,z} (axes may be negative = direction),
//	    BlockStatePalette: List<{Name, Properties}> (air always index 0),
//	    BlockStates: long array — bits = max(2, ceil(log2(paletteSize))),
//	                 entries packed contiguously, spanning long boundaries,
//	    TileEntities (vanilla-style, x/y/z inline), Entities,
//	    PendingBlockTicks, PendingFluidTicks
//	  }}
//	}
//
// Region block order is x-fastest: index = x + sizeX*(z + sizeZ*y) — the
// model's native layout.

type litVec3 struct {
	X int32 `nbt:"x"`
	Y int32 `nbt:"y"`
	Z int32 `nbt:"z"`
}

type litPaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type litRegionIn struct {
	Position          litVec3           `nbt:"Position"`
	Size              litVec3           `nbt:"Size"`
	BlockStatePalette []litPaletteEntry `nbt:"BlockStatePalette"`
	BlockStates       []int64           `nbt:"BlockStates"`
	TileEntities      []nbt.RawMessage  `nbt:"TileEntities"`
	Entities          []nbt.RawMessage  `nbt:"Entities"`
	PendingBlockTicks []nbt.RawMessage  `nbt:"PendingBlockTicks"`
	PendingFluidTicks []nbt.RawMessage  `nbt:"PendingFluidTicks"`
}

type litMetadataIn struct {
	Name          string  `nbt:"Name"`
	Author        string  `nbt:"Author"`
	EnclosingSize litVec3 `nbt:"EnclosingSize"`
}

type litRootIn struct {
	MinecraftDataVersion int32                  `nbt:"MinecraftDataVersion"`
	Version              int32                  `nbt:"Version"`
	Metadata             litMetadataIn          `nbt:"Metadata"`
	Regions              map[string]litRegionIn `nbt:"Regions"`
}

// litBitsFor returns the packed bits per entry for a palette size.
func litBitsFor(paletteSize int) int {
	b := bits.Len(uint(paletteSize - 1))
	if b < 2 {
		b = 2
	}
	return b
}

// litUnpack reads entry i from a spanning-packed long array.
func litUnpack(longs []int64, bitsPer, i int) int {
	bitPos := i * bitsPer
	li := bitPos >> 6
	off := bitPos & 63
	mask := uint64(1)<<bitsPer - 1
	v := uint64(longs[li]) >> off
	if off+bitsPer > 64 {
		v |= uint64(longs[li+1]) << (64 - off)
	}
	return int(v & mask)
}

// litPack builds a spanning-packed long array from palette indices.
func litPack(values []int32, bitsPer int) []int64 {
	total := (len(values)*bitsPer + 63) / 64
	longs := make([]uint64, total)
	for i, v := range values {
		bitPos := i * bitsPer
		li := bitPos >> 6
		off := bitPos & 63
		longs[li] |= uint64(v) << off
		if off+bitsPer > 64 {
			longs[li+1] |= uint64(v) >> (64 - off)
		}
	}
	out := make([]int64, total)
	for i, u := range longs {
		out[i] = int64(u)
	}
	return out
}

// ReadLitematic parses a .litematic into the normalized model. Multiple
// regions are flattened into their enclosing bounding box (lossy-noted).
func ReadLitematic(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root litRootIn
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not a valid .litematic: %w", err)
	}
	if len(root.Regions) == 0 {
		return nil, fmt.Errorf("schematic: .litematic has no regions")
	}

	// Normalize each region: negative Size axes grow in the negative
	// direction from Position.
	type normRegion struct {
		name          string
		min           [3]int
		size          [3]int
		palette       []litPaletteEntry
		states        []int64
		tileEntities  []nbt.RawMessage
		entities      []nbt.RawMessage
		droppedTicks  bool
	}
	var regions []normRegion
	for name, r := range root.Regions {
		pos := [3]int32{r.Position.X, r.Position.Y, r.Position.Z}
		size := [3]int32{r.Size.X, r.Size.Y, r.Size.Z}
		var n normRegion
		n.name = name
		for a := 0; a < 3; a++ {
			if size[a] == 0 {
				return nil, fmt.Errorf("schematic: .litematic region %q has zero size axis", name)
			}
			if size[a] < 0 {
				n.min[a] = int(pos[a] + size[a] + 1)
				n.size[a] = int(-size[a])
			} else {
				n.min[a] = int(pos[a])
				n.size[a] = int(size[a])
			}
			if n.size[a] > MaxDimension {
				return nil, fmt.Errorf("schematic: .litematic region %q exceeds maximum dimension", name)
			}
		}
		if v := n.size[0] * n.size[1] * n.size[2]; v > MaxVolume {
			return nil, fmt.Errorf("schematic: .litematic region %q volume exceeds maximum", name)
		}
		if len(r.BlockStatePalette) == 0 || len(r.BlockStatePalette) > MaxPaletteSize {
			return nil, fmt.Errorf("schematic: .litematic region %q has invalid palette size %d", name, len(r.BlockStatePalette))
		}
		bitsPer := litBitsFor(len(r.BlockStatePalette))
		need := (n.size[0]*n.size[1]*n.size[2]*bitsPer + 63) / 64
		if len(r.BlockStates) < need {
			return nil, fmt.Errorf("schematic: .litematic region %q block states truncated (%d longs, need %d)", name, len(r.BlockStates), need)
		}
		n.palette = r.BlockStatePalette
		n.states = r.BlockStates
		n.tileEntities = r.TileEntities
		n.entities = r.Entities
		n.droppedTicks = len(r.PendingBlockTicks) > 0 || len(r.PendingFluidTicks) > 0
		regions = append(regions, n)
	}

	// Enclosing bounding box across all regions.
	min := regions[0].min
	max := [3]int{}
	for a := 0; a < 3; a++ {
		max[a] = regions[0].min[a] + regions[0].size[a]
	}
	for _, r := range regions[1:] {
		for a := 0; a < 3; a++ {
			if r.min[a] < min[a] {
				min[a] = r.min[a]
			}
			if end := r.min[a] + r.size[a]; end > max[a] {
				max[a] = end
			}
		}
	}
	sx, sy, sz := max[0]-min[0], max[1]-min[1], max[2]-min[2]
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: .litematic enclosing size %dx%dx%d exceeds maximum", sx, sy, sz)
	}
	if v := sx * sy * sz; v > MaxVolume {
		return nil, fmt.Errorf("schematic: .litematic enclosing volume %d exceeds maximum", v)
	}

	s := New(sx, sy, sz)
	s.DataVersion = int(root.MinecraftDataVersion)
	s.Meta.SourceFormat = "litematic"
	s.Meta.Name = root.Metadata.Name
	s.Meta.Author = root.Metadata.Author
	if len(regions) > 1 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf(".litematic has %d regions; flattened into one bounding box", len(regions)))
	}

	beTotal := 0
	for _, r := range regions {
		if r.droppedTicks {
			s.Meta.LossyNotes = append(s.Meta.LossyNotes,
				fmt.Sprintf("region %q pending block/fluid ticks dropped", r.name))
		}
		bitsPer := litBitsFor(len(r.palette))
		// Region palette → model palette.
		srcToModel := make([]int32, len(r.palette))
		for i, e := range r.palette {
			if len(e.Name) == 0 || len(e.Name) > MaxBlockIDLength {
				return nil, fmt.Errorf("schematic: .litematic palette entry %d invalid", i)
			}
			srcToModel[i] = s.PaletteIndex(BlockState{Name: e.Name, Properties: e.Properties})
		}
		ox, oy, oz := r.min[0]-min[0], r.min[1]-min[1], r.min[2]-min[2]
		i := 0
		for y := 0; y < r.size[1]; y++ {
			for z := 0; z < r.size[2]; z++ {
				for x := 0; x < r.size[0]; x++ {
					v := litUnpack(r.states, bitsPer, i)
					i++
					if v >= len(srcToModel) {
						return nil, fmt.Errorf("schematic: .litematic block references palette index %d out of range", v)
					}
					s.Blocks[s.Index(ox+x, oy+y, oz+z)] = srcToModel[v]
				}
			}
		}

		// Tile entities: vanilla-style with inline x/y/z (region-relative).
		for _, teRaw := range r.tileEntities {
			beTotal++
			if beTotal > MaxBlockEntities {
				return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
			}
			fields, err := compoundFields(teRaw)
			if err != nil {
				return nil, fmt.Errorf("schematic: .litematic tile entity: %w", err)
			}
			px, okx := intFromRaw(fields["x"])
			py, oky := intFromRaw(fields["y"])
			pz, okz := intFromRaw(fields["z"])
			if !okx || !oky || !okz {
				return nil, fmt.Errorf("schematic: .litematic tile entity missing x/y/z")
			}
			delete(fields, "x")
			delete(fields, "y")
			delete(fields, "z")
			bx, by, bz := ox+int(px), oy+int(py), oz+int(pz)
			if bx < 0 || bx >= sx || by < 0 || by >= sy || bz < 0 || bz >= sz {
				return nil, fmt.Errorf("schematic: .litematic tile entity outside bounds")
			}
			s.BlockEntities = append(s.BlockEntities, BlockEntity{
				Pos: [3]int{bx, by, bz},
				Raw: compoundFromFields(fields),
			})
		}
		s.Entities = append(s.Entities, r.entities...)
	}
	if len(s.Entities) > MaxEntities {
		return nil, fmt.Errorf("schematic: more than %d entities", MaxEntities)
	}
	if len(s.Entities) > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes, "entity coordinate spaces differ between formats; entities carried as-is")
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

type litRegionOut struct {
	Position          litVec3           `nbt:"Position"`
	Size              litVec3           `nbt:"Size"`
	BlockStatePalette []litPaletteEntryOut `nbt:"BlockStatePalette"`
	BlockStates       []int64           `nbt:"BlockStates"`
	TileEntities      rawList          `nbt:"TileEntities"`
	Entities          rawList          `nbt:"Entities"`
}

type litPaletteEntryOut struct {
	Name       string   `nbt:"Name"`
	Properties propsMap `nbt:"Properties,omitempty"`
}

type litMetadataOut struct {
	Name          string  `nbt:"Name"`
	Author        string  `nbt:"Author"`
	Description   string  `nbt:"Description"`
	EnclosingSize litVec3 `nbt:"EnclosingSize"`
	RegionCount   int32   `nbt:"RegionCount"`
	TimeCreated   int64   `nbt:"TimeCreated"`
	TimeModified  int64   `nbt:"TimeModified"`
	TotalBlocks   int32   `nbt:"TotalBlocks"`
	TotalVolume   int32   `nbt:"TotalVolume"`
}

type litRootOut struct {
	MinecraftDataVersion int32                   `nbt:"MinecraftDataVersion"`
	Version              int32                   `nbt:"Version"`
	Metadata             litMetadataOut          `nbt:"Metadata"`
	Regions              map[string]litRegionOut `nbt:"Regions"`
}

// WriteLitematic serializes the model as a gzip-compressed .litematic
// (format version 6) with a single region. Timestamps are zero so output is
// deterministic and cache-friendly.
func WriteLitematic(s *Schematic) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	// Litematica palettes put air at index 0; the model already guarantees
	// that (New seeds air first), but compact unused entries while keeping
	// air at 0.
	used := make([]bool, len(s.Palette))
	used[0] = true // air stays
	for _, idx := range s.Blocks {
		used[idx] = true
	}
	modelToOut := make([]int32, len(s.Palette))
	var outPalette []litPaletteEntryOut
	for i, st := range s.Palette {
		if !used[i] {
			modelToOut[i] = -1
			continue
		}
		modelToOut[i] = int32(len(outPalette))
		outPalette = append(outPalette, litPaletteEntryOut{Name: st.Name, Properties: st.Properties})
	}
	values := make([]int32, len(s.Blocks))
	for i, idx := range s.Blocks {
		values[i] = modelToOut[idx]
	}
	bitsPer := litBitsFor(len(outPalette))

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
	entities := rawList(s.Entities)

	name := s.Meta.Name
	if name == "" {
		name = "Main"
	}
	size := litVec3{int32(s.Size[0]), int32(s.Size[1]), int32(s.Size[2])}
	root := litRootOut{
		MinecraftDataVersion: int32(s.DataVersion),
		Version:              6,
		Metadata: litMetadataOut{
			Name:          name,
			Author:        s.Meta.Author,
			EnclosingSize: size,
			RegionCount:   1,
			TotalBlocks:   int32(s.BlockCount()),
			TotalVolume:   int32(s.Volume()),
		},
		Regions: map[string]litRegionOut{
			name: {
				Position:          litVec3{0, 0, 0},
				Size:              size,
				BlockStatePalette: outPalette,
				BlockStates:       litPack(values, bitsPer),
				TileEntities:      tiles,
				Entities:          entities,
			},
		},
	}

	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, ""); err != nil {
		return nil, fmt.Errorf("schematic: encode .litematic: %w", err)
	}
	return gzipBytes(buf.Bytes())
}
