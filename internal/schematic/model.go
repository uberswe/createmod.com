// Package schematic is the normalized schematic library: every supported
// file format parses into one internal model — palette + block index array +
// block entities + entities + DataVersion — and serializes back out. It is
// the single dependency under format conversion, the editor, calculators,
// material lists, the NBT viewer and world export.
//
// The model is deliberately palette-based: vanilla structure NBT, Sponge
// .schem, Litematica regions and Anvil chunk sections all store blocks as
// palette indices, so this representation converts to and from each of them
// without loss.
package schematic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// BlockState is one palette entry: a block name plus its blockstate
// properties.
type BlockState struct {
	Name       string
	Properties map[string]string
}

// Key returns a canonical string form (sorted properties) used for palette
// deduplication.
func (b BlockState) Key() string {
	if len(b.Properties) == 0 {
		return b.Name
	}
	keys := make([]string, 0, len(b.Properties))
	for k := range b.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString(b.Name)
	sb.WriteByte('[')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(b.Properties[k])
	}
	sb.WriteByte(']')
	return sb.String()
}

// IsAir reports whether the state is plain air (not cave/void air, which
// occupy space deliberately).
func (b BlockState) IsAir() bool { return b.Name == "minecraft:air" }

// BlockEntity carries a block entity's position and its raw NBT payload.
// The payload is kept opaque so Create kinetic data (and any modded data)
// round-trips byte-faithfully without this package modeling it.
type BlockEntity struct {
	Pos [3]int
	Raw nbt.RawMessage
}

// Meta carries provenance and fidelity information about the model.
type Meta struct {
	// Name is a display name when the source format carries one.
	Name string
	// Author when the source format carries one.
	Author string
	// SourceFormat is the format the model was read from ("structure",
	// "schem", "litematic", "schematic", "sable").
	SourceFormat string
	// LossyNotes records fidelity losses incurred while reading (e.g.
	// "multiple random palettes flattened to the first").
	LossyNotes []string
}

// Schematic is the normalized model.
type Schematic struct {
	// Size is the bounding box (X, Y, Z), always positive.
	Size [3]int
	// Palette holds the distinct block states. Index 0 is guaranteed to be
	// minecraft:air after ReadStructureNBT / Normalize.
	Palette []BlockState
	// Blocks holds one palette index per position, laid out X-major:
	// index = x + Size[0]*(z + Size[2]*y). Length = Size[0]*Size[1]*Size[2].
	Blocks []int32
	// BlockEntities in schematic-local coordinates.
	BlockEntities []BlockEntity
	// Entities as raw NBT list entries (position data embedded per format).
	Entities []nbt.RawMessage
	// DataVersion of the source; 0 means unknown (legacy formats).
	DataVersion int
	Meta        Meta
}

// Index converts (x, y, z) to a Blocks offset. Callers must bounds-check.
func (s *Schematic) Index(x, y, z int) int {
	return x + s.Size[0]*(z+s.Size[2]*y)
}

// At returns the palette entry at (x, y, z).
func (s *Schematic) At(x, y, z int) BlockState {
	return s.Palette[s.Blocks[s.Index(x, y, z)]]
}

// Volume returns the bounding-box volume.
func (s *Schematic) Volume() int { return s.Size[0] * s.Size[1] * s.Size[2] }

// Validate checks internal consistency (sizes, palette indices in range).
func (s *Schematic) Validate() error {
	if s.Size[0] <= 0 || s.Size[1] <= 0 || s.Size[2] <= 0 {
		return fmt.Errorf("schematic: non-positive size %v", s.Size)
	}
	if s.Size[0] > MaxDimension || s.Size[1] > MaxDimension || s.Size[2] > MaxDimension {
		return fmt.Errorf("schematic: size %v exceeds maximum dimension %d", s.Size, MaxDimension)
	}
	if len(s.Palette) == 0 {
		return fmt.Errorf("schematic: empty palette")
	}
	if len(s.Palette) > MaxPaletteSize {
		return fmt.Errorf("schematic: palette size %d exceeds maximum %d", len(s.Palette), MaxPaletteSize)
	}
	if want := s.Volume(); len(s.Blocks) != want {
		return fmt.Errorf("schematic: blocks length %d does not match volume %d", len(s.Blocks), want)
	}
	n := int32(len(s.Palette))
	for i, idx := range s.Blocks {
		if idx < 0 || idx >= n {
			return fmt.Errorf("schematic: block %d has palette index %d out of range", i, idx)
		}
	}
	for _, be := range s.BlockEntities {
		for a := 0; a < 3; a++ {
			if be.Pos[a] < 0 || be.Pos[a] >= s.Size[a] {
				return fmt.Errorf("schematic: block entity at %v outside bounds %v", be.Pos, s.Size)
			}
		}
	}
	return nil
}

// PaletteIndex returns the palette index for a state, appending it if new.
func (s *Schematic) PaletteIndex(b BlockState) int32 {
	key := b.Key()
	for i, p := range s.Palette {
		if p.Key() == key {
			return int32(i)
		}
	}
	s.Palette = append(s.Palette, b)
	return int32(len(s.Palette) - 1)
}

// New returns an all-air schematic of the given size with air at palette 0.
func New(sx, sy, sz int) *Schematic {
	s := &Schematic{
		Size:    [3]int{sx, sy, sz},
		Palette: []BlockState{{Name: "minecraft:air"}},
		Blocks:  make([]int32, sx*sy*sz),
	}
	return s
}
