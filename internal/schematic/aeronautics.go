package schematic

import (
	"fmt"
	"strconv"

	"github.com/Tnze/go-mc/nbt"
)

// Create: Aeronautics Toolgun blueprint (.excraft), enxv's native format
// (com.enxv.aeronauticsstructuretool). The toolgun saves a craft as raw
// Minecraft chunk sections (Anvil world data) grouped into one or more
// sub-levels, gzip-compressed:
//
//	root {
//	  format: String ("enxv_aeronautics_plot_print_v8" | "_v9")
//	  name: String
//	  sublevels: List<Compound>{
//	    plot: {
//	      chunks: Compound keyed by ChunkPos.asLong (string) {
//	        <chunkLong>: {
//	          sections: Compound keyed by section-Y (string) {
//	            <sectionY>: { block_states: { palette: List<{Name,Properties}>,
//	                          data: LongArray } }   // MC 1.18 non-spanning
//	          }
//	          block_entities: List<Compound> (world x/y/z inline)
//	        }
//	      }
//	    }
//	  }
//	}
//
// Reading flattens every sub-level's chunk sections into the model at their
// saved world coordinates and drops what a static schematic can't carry:
// entities, Create contraption runtime state, sub-level relative poses /
// orientation, previews and lighting (recorded as lossy notes). Writing
// .excraft is not supported.

const (
	formatExcraftV8     = "enxv_aeronautics_plot_print_v8"
	formatExcraftV9     = "enxv_aeronautics_plot_print_v9"
	formatExcraftPrefix = "enxv_aeronautics"
)

type excraftPaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type excraftBlockStates struct {
	Palette []excraftPaletteEntry `nbt:"palette"`
	Data    []int64               `nbt:"data"`
}

type excraftSection struct {
	BlockStates excraftBlockStates `nbt:"block_states"`
}

type excraftChunk struct {
	Sections      map[string]excraftSection `nbt:"sections"`
	BlockEntities []nbt.RawMessage          `nbt:"block_entities"`
}

type excraftSubLevel struct {
	Plot struct {
		Chunks map[string]excraftChunk `nbt:"chunks"`
	} `nbt:"plot"`
}

type excraftRoot struct {
	Format               string            `nbt:"format"`
	Name                 string            `nbt:"name"`
	SourceMinBuildHeight int32             `nbt:"source_min_build_height"`
	SubLevels            []excraftSubLevel `nbt:"sublevels"`
}

// excraftIsFormat reports whether a decoded root "format" string is a
// supported Aeronautics Toolgun blueprint version.
func excraftIsFormat(format string) bool {
	return format == formatExcraftV8 || format == formatExcraftV9
}

// aeroSectionBits returns the bits-per-entry for a chunk section palette:
// at least 4, otherwise the smallest width that indexes the palette. Chunk
// sections (unlike Litematica) do not span long boundaries.
func aeroSectionBits(paletteLen int) int {
	bits := 4
	for (1 << bits) < paletteLen {
		bits++
	}
	return bits
}

// aeroDecodeSection unpacks a section's 4096 palette indices (MC 1.18
// non-spanning layout). A palette of one entry (or no data) is a solid
// section of that single block.
func aeroDecodeSection(bs excraftBlockStates) ([]BlockState, []int32) {
	pal := make([]BlockState, len(bs.Palette))
	for i, e := range bs.Palette {
		pal[i] = BlockState{Name: e.Name, Properties: e.Properties}
	}
	idx := make([]int32, 4096)
	if len(pal) <= 1 || len(bs.Data) == 0 {
		return pal, idx // all zero -> palette[0]
	}
	bits := aeroSectionBits(len(pal))
	perLong := 64 / bits
	mask := int64(1)<<uint(bits) - 1
	for i := 0; i < 4096; i++ {
		li := i / perLong
		if li >= len(bs.Data) {
			break
		}
		off := uint((i % perLong) * bits)
		idx[i] = int32((bs.Data[li] >> off) & mask)
	}
	return pal, idx
}

// aeroChunkXZ decodes a ChunkPos.asLong key into chunk x/z.
func aeroChunkXZ(key string) (int, int, bool) {
	cl, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return int(int32(uint64(cl) & 0xFFFFFFFF)), int(int32(uint64(cl) >> 32)), true
}

type aeroBlock struct {
	pos   [3]int
	state BlockState
}

// ReadExcraft parses a Create: Aeronautics Toolgun blueprint into the model.
func ReadExcraft(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root excraftRoot
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not an Aeronautics blueprint: %w", err)
	}
	if !excraftIsFormat(root.Format) {
		return nil, fmt.Errorf("schematic: unsupported Aeronautics blueprint format %q", root.Format)
	}

	// Section keys are stored offset from the world bottom (non-negative), so
	// worldSectionY = key + minSection. Block entities carry absolute world
	// coordinates, so blocks must use the same absolute Y or they won't align.
	minH := int(root.SourceMinBuildHeight)
	if minH == 0 {
		minH = -64 // MC 1.18+ / the v8 format's LEGACY_V8_MIN_BUILD_HEIGHT
	}
	secOffset := minH / 16

	var (
		blocks  []aeroBlock
		beWorld []BlockEntity // positions still in world space
		notes   []string
		minSet  bool
		min     [3]int
		max     [3]int
	)
	extend := func(x, y, z int) {
		if !minSet {
			min = [3]int{x, y, z}
			max = [3]int{x, y, z}
			minSet = true
			return
		}
		for a, v := range [3]int{x, y, z} {
			if v < min[a] {
				min[a] = v
			}
			if v > max[a] {
				max[a] = v
			}
		}
	}

	for _, sl := range root.SubLevels {
		for chunkKey, chunk := range sl.Plot.Chunks {
			cx, cz, ok := aeroChunkXZ(chunkKey)
			if !ok {
				continue
			}
			for secKey, sec := range chunk.Sections {
				sy, err := strconv.Atoi(secKey)
				if err != nil {
					continue
				}
				pal, idx := aeroDecodeSection(sec.BlockStates)
				if len(pal) == 0 {
					continue
				}
				for i := 0; i < 4096; i++ {
					pi := idx[i]
					if int(pi) >= len(pal) {
						continue
					}
					st := pal[pi]
					if st.IsAir() {
						continue
					}
					lx, lz, ly := i&15, (i>>4)&15, (i>>8)&15
					wx, wy, wz := cx*16+lx, (sy+secOffset)*16+ly, cz*16+lz
					blocks = append(blocks, aeroBlock{pos: [3]int{wx, wy, wz}, state: st})
					extend(wx, wy, wz)
				}
			}
			for _, be := range chunk.BlockEntities {
				var pos struct {
					X int32 `nbt:"x"`
					Y int32 `nbt:"y"`
					Z int32 `nbt:"z"`
				}
				if err := unmarshalRaw(be, &pos); err != nil {
					continue
				}
				// Strip the world x/y/z so the payload matches structure-NBT
				// block entities (position comes from the block's slot).
				clean := be
				if fields, err := compoundFields(be); err == nil {
					delete(fields, "x")
					delete(fields, "y")
					delete(fields, "z")
					clean = compoundFromFields(fields)
				}
				beWorld = append(beWorld, BlockEntity{Pos: [3]int{int(pos.X), int(pos.Y), int(pos.Z)}, Raw: clean})
			}
		}
	}

	if !minSet || len(blocks) == 0 {
		return nil, fmt.Errorf("schematic: Aeronautics blueprint has no blocks")
	}

	sx := max[0] - min[0] + 1
	sy := max[1] - min[1] + 1
	sz := max[2] - min[2] + 1
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: Aeronautics blueprint size %dx%dx%d exceeds maximum dimension %d", sx, sy, sz, MaxDimension)
	}

	s := New(sx, sy, sz) // all-air, palette[0] = minecraft:air
	for _, b := range blocks {
		x, y, z := b.pos[0]-min[0], b.pos[1]-min[1], b.pos[2]-min[2]
		s.Blocks[s.Index(x, y, z)] = s.PaletteIndex(b.state)
	}
	for _, be := range beWorld {
		x, y, z := be.Pos[0]-min[0], be.Pos[1]-min[1], be.Pos[2]-min[2]
		if x < 0 || y < 0 || z < 0 || x >= sx || y >= sy || z >= sz {
			continue
		}
		s.BlockEntities = append(s.BlockEntities, BlockEntity{Pos: [3]int{x, y, z}, Raw: be.Raw})
	}

	if len(root.SubLevels) > 1 {
		notes = append(notes, fmt.Sprintf("flattened %d sub-levels into one structure at their saved world positions", len(root.SubLevels)))
	}
	notes = append(notes, "entities, contraption runtime state, sub-level poses/orientation, previews and lighting were dropped (blocks and block entities only)")

	s.DataVersion = 3955 // Create Aeronautics targets MC 1.21.1
	s.Meta = Meta{
		Name:         root.Name,
		SourceFormat: string(FormatExcraft),
		LossyNotes:   notes,
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}
