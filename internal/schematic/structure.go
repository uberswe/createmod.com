package schematic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/Tnze/go-mc/nbt"
)

// Vanilla structure NBT (the format Create schematics use):
//
//	root {
//	  size: TAG_List(Int)[3]
//	  palette: TAG_List(Compound){ Name, Properties? }
//	  palettes: optional TAG_List of palettes (random variants)
//	  blocks: TAG_List(Compound){ pos: TAG_List(Int)[3], state: Int, nbt? }
//	  entities: TAG_List(Compound){ pos, blockPos, nbt }
//	  DataVersion: Int
//	}

// intList encodes as TAG_List of TAG_Int instead of TAG_Int_Array. The Tnze
// encoder writes []int32 as TAG_Int_Array, but Minecraft's structure format
// (and Create's parser) require TAG_List for pos and size fields — files
// written without this are rejected in-game.
type intList []int32

func (l intList) TagType() byte { return nbt.TagList }

func (l intList) MarshalNBT(w io.Writer) error {
	if _, err := w.Write([]byte{nbt.TagInt}); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, int32(len(l))); err != nil {
		return err
	}
	for _, v := range l {
		if err := binary.Write(w, binary.BigEndian, v); err != nil {
			return err
		}
	}
	return nil
}

type structPaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type structBlockIn struct {
	Pos   []int32        `nbt:"pos"`
	State int32          `nbt:"state"`
	NBT   nbt.RawMessage `nbt:"nbt"`
}

type structRootIn struct {
	DataVersion int32                `nbt:"DataVersion"`
	Size        []int32              `nbt:"size"`
	Palette     []structPaletteEntry `nbt:"palette"`
	Palettes    [][]structPaletteEntry `nbt:"palettes"`
	Blocks      []structBlockIn      `nbt:"blocks"`
	Entities    []nbt.RawMessage     `nbt:"entities"`
	Author      string               `nbt:"author"`
}

// rawNBT re-emits a decoded nbt.RawMessage payload byte-for-byte; the Tnze
// encoder cannot marshal RawMessage values directly.
type rawNBT nbt.RawMessage

func (r rawNBT) TagType() byte { return r.Type }

func (r rawNBT) MarshalNBT(w io.Writer) error {
	_, err := w.Write(r.Data)
	return err
}

type structBlockOut struct {
	Pos   intList `nbt:"pos"`
	State int32   `nbt:"state"`
}

type structBlockOutBE struct {
	Pos   intList `nbt:"pos"`
	State int32   `nbt:"state"`
	NBT   rawNBT  `nbt:"nbt"`
}

// propsMap marshals blockstate properties with sorted keys so writer output
// is byte-stable (Go map iteration order would otherwise randomize files,
// defeating cache keys derived from content).
type propsMap map[string]string

func (p propsMap) TagType() byte { return nbt.TagCompound }

func (p propsMap) MarshalNBT(w io.Writer) error {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	writeStr := func(s string) error {
		if err := binary.Write(w, binary.BigEndian, uint16(len(s))); err != nil {
			return err
		}
		_, err := io.WriteString(w, s)
		return err
	}
	for _, k := range keys {
		if _, err := w.Write([]byte{nbt.TagString}); err != nil {
			return err
		}
		if err := writeStr(k); err != nil {
			return err
		}
		if err := writeStr(p[k]); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte{nbt.TagEnd})
	return err
}

type structPaletteEntryOut struct {
	Name       string   `nbt:"Name"`
	Properties propsMap `nbt:"Properties,omitempty"`
}

type structRootOut struct {
	DataVersion int32                   `nbt:"DataVersion"`
	Size        intList                 `nbt:"size"`
	Palette     []structPaletteEntryOut `nbt:"palette"`
	Blocks      []interface{}           `nbt:"blocks"`
	Entities    rawList                `nbt:"entities"`
}

// ReadStructureNBT parses a vanilla structure / Create schematic file
// (gzip, zlib or uncompressed) into the normalized model.
func ReadStructureNBT(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root structRootIn
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not valid structure NBT: %w", err)
	}

	if len(root.Size) != 3 {
		return nil, fmt.Errorf("schematic: structure size has %d elements, want 3", len(root.Size))
	}
	sx, sy, sz := int(root.Size[0]), int(root.Size[1]), int(root.Size[2])
	if sx <= 0 || sy <= 0 || sz <= 0 {
		return nil, fmt.Errorf("schematic: non-positive structure size %dx%dx%d", sx, sy, sz)
	}
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: structure size %dx%dx%d exceeds maximum dimension", sx, sy, sz)
	}
	if v := sx * sy * sz; v > MaxVolume {
		return nil, fmt.Errorf("schematic: structure volume %d exceeds maximum %d", v, MaxVolume)
	}

	s := New(sx, sy, sz)
	s.DataVersion = int(root.DataVersion)
	s.Meta.SourceFormat = "structure"
	s.Meta.Author = root.Author

	// Random palette variants ("palettes") are flattened to the first.
	srcPalette := root.Palette
	if len(srcPalette) == 0 && len(root.Palettes) > 0 {
		srcPalette = root.Palettes[0]
		if len(root.Palettes) > 1 {
			s.Meta.LossyNotes = append(s.Meta.LossyNotes,
				fmt.Sprintf("structure has %d random palette variants; flattened to the first", len(root.Palettes)))
		}
	}
	if len(srcPalette) > MaxPaletteSize {
		return nil, fmt.Errorf("schematic: palette size %d exceeds maximum %d", len(srcPalette), MaxPaletteSize)
	}

	// Map source palette indices to model palette indices (model index 0 is
	// always air; source palettes may or may not contain air).
	srcToModel := make([]int32, len(srcPalette))
	for i, e := range srcPalette {
		if len(e.Name) == 0 || len(e.Name) > MaxBlockIDLength {
			return nil, fmt.Errorf("schematic: palette entry %d has invalid block id", i)
		}
		srcToModel[i] = s.PaletteIndex(BlockState{Name: e.Name, Properties: e.Properties})
	}

	if len(root.Blocks) > sx*sy*sz {
		return nil, fmt.Errorf("schematic: %d block entries exceed structure volume", len(root.Blocks))
	}
	beCount := 0
	for i, b := range root.Blocks {
		if len(b.Pos) != 3 {
			return nil, fmt.Errorf("schematic: block %d has %d pos elements, want 3", i, len(b.Pos))
		}
		x, y, z := int(b.Pos[0]), int(b.Pos[1]), int(b.Pos[2])
		if x < 0 || x >= sx || y < 0 || y >= sy || z < 0 || z >= sz {
			return nil, fmt.Errorf("schematic: block %d at (%d,%d,%d) outside bounds", i, x, y, z)
		}
		if b.State < 0 || int(b.State) >= len(srcToModel) {
			return nil, fmt.Errorf("schematic: block %d references palette state %d out of range", i, b.State)
		}
		s.Blocks[s.Index(x, y, z)] = srcToModel[b.State]
		if len(b.NBT.Data) > 0 {
			beCount++
			if beCount > MaxBlockEntities {
				return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
			}
			s.BlockEntities = append(s.BlockEntities, BlockEntity{Pos: [3]int{x, y, z}, Raw: b.NBT})
		}
	}

	if len(root.Entities) > MaxEntities {
		return nil, fmt.Errorf("schematic: more than %d entities", MaxEntities)
	}
	s.Entities = root.Entities

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

// WriteStructureNBT serializes the model as a gzip-compressed vanilla
// structure NBT file. Air blocks are omitted from the sparse block list
// (the in-game loader treats absent positions as air), and block entities
// are re-embedded on their block entries.
func WriteStructureNBT(s *Schematic) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	// Block entities by position for embedding.
	beAt := make(map[[3]int]nbt.RawMessage, len(s.BlockEntities))
	for _, be := range s.BlockEntities {
		beAt[be.Pos] = be.Raw
	}

	// Compact the palette to states actually used by non-air blocks (plus
	// air itself if any block entity sits on air, which is invalid anyway).
	used := make([]bool, len(s.Palette))
	for _, idx := range s.Blocks {
		used[idx] = true
	}
	modelToOut := make([]int32, len(s.Palette))
	var outPalette []structPaletteEntryOut
	for i, st := range s.Palette {
		if !used[i] || st.IsAir() {
			modelToOut[i] = -1
			continue
		}
		modelToOut[i] = int32(len(outPalette))
		outPalette = append(outPalette, structPaletteEntryOut{Name: st.Name, Properties: st.Properties})
	}

	var blocks []interface{}
	for y := 0; y < s.Size[1]; y++ {
		for z := 0; z < s.Size[2]; z++ {
			for x := 0; x < s.Size[0]; x++ {
				outIdx := modelToOut[s.Blocks[s.Index(x, y, z)]]
				if outIdx < 0 {
					continue
				}
				pos := intList{int32(x), int32(y), int32(z)}
				if raw, ok := beAt[[3]int{x, y, z}]; ok {
					blocks = append(blocks, structBlockOutBE{Pos: pos, State: outIdx, NBT: rawNBT(raw)})
				} else {
					blocks = append(blocks, structBlockOut{Pos: pos, State: outIdx})
				}
			}
		}
	}

	entities := rawList(s.Entities)
	root := structRootOut{
		DataVersion: int32(s.DataVersion),
		Size:        intList{int32(s.Size[0]), int32(s.Size[1]), int32(s.Size[2])},
		Palette:     outPalette,
		Blocks:      blocks,
		Entities:    entities,
	}
	if root.Palette == nil {
		root.Palette = []structPaletteEntryOut{}
	}
	if root.Blocks == nil {
		root.Blocks = []interface{}{}
	}

	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(root, ""); err != nil {
		return nil, fmt.Errorf("schematic: encode structure NBT: %w", err)
	}
	return gzipBytes(buf.Bytes())
}
