package schematic

import (
	"fmt"

	"github.com/Tnze/go-mc/nbt"
)

// Sable Schematic Tool Blueprint v1 (dev.rew1nd.sableschematicapi):
//
//	root {
//	  version: Int (1)
//	  origin: {x,y,z doubles}
//	  canonical_bound | root_bounds: {min_x..max_z}
//	  global_extra_data: Compound
//	  preview: optional Compound
//	  sub_levels: List<Compound> [{
//	    id: Int, source_uuid, relative_pose, local_bounds,
//	    blocks_origin: {x,y,z ints},
//	    block_palette: List<{Name, Properties}> (vanilla BlockState codec),
//	    unavailable_palette_ids: optional IntArray,
//	    blocks: List<{local_pos: {x,y,z}, palette_id, block_entity_data_id?}>,
//	    block_entities: List<Compound>, entities: List, extra_data, name?
//	  }]
//	}
//
// Reading is a deliberate blocks-only flatten: sub-level poses, entities
// (including Create contraptions), previews and extra data are dropped with
// lossy notes. Writing is not supported while the format is experimental.

type sableVec3i struct {
	X int32 `nbt:"x"`
	Y int32 `nbt:"y"`
	Z int32 `nbt:"z"`
}

type sableBlockIn struct {
	LocalPos          sableVec3i `nbt:"local_pos"`
	PaletteID         int32      `nbt:"palette_id"`
	BlockEntityDataID *int32     `nbt:"block_entity_data_id"`
}

type sablePaletteEntry struct {
	Name       string            `nbt:"Name"`
	Properties map[string]string `nbt:"Properties,omitempty"`
}

type sableSubLevelIn struct {
	ID                    int32               `nbt:"id"`
	Name                  string              `nbt:"name"`
	BlocksOrigin          sableVec3i          `nbt:"blocks_origin"`
	BlockPalette          []sablePaletteEntry `nbt:"block_palette"`
	UnavailablePaletteIDs []int32             `nbt:"unavailable_palette_ids"`
	Blocks                []sableBlockIn      `nbt:"blocks"`
	BlockEntities         []nbt.RawMessage    `nbt:"block_entities"`
	Entities              []nbt.RawMessage    `nbt:"entities"`
}

type sableRootIn struct {
	Version   int32             `nbt:"version"`
	SubLevels []sableSubLevelIn `nbt:"sub_levels"`
}

// ReadSable parses a Sable Blueprint v1 into the normalized model
// (blocks-only flatten).
func ReadSable(data []byte) (*Schematic, error) {
	raw, err := decompress(data)
	if err != nil {
		return nil, err
	}
	var root sableRootIn
	if err := nbt.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("schematic: not a valid Sable blueprint: %w", err)
	}
	if root.Version != 1 {
		return nil, fmt.Errorf("schematic: unsupported Sable blueprint version %d", root.Version)
	}
	if len(root.SubLevels) == 0 {
		return nil, fmt.Errorf("schematic: Sable blueprint has no sub-levels")
	}

	// Bounding box across all sub-levels in blocks_origin space.
	first := true
	var min, max [3]int
	totalBlocks := 0
	for _, sl := range root.SubLevels {
		totalBlocks += len(sl.Blocks)
		if totalBlocks > MaxVolume {
			return nil, fmt.Errorf("schematic: Sable blueprint block count exceeds maximum")
		}
		for _, b := range sl.Blocks {
			p := [3]int{
				int(sl.BlocksOrigin.X + b.LocalPos.X),
				int(sl.BlocksOrigin.Y + b.LocalPos.Y),
				int(sl.BlocksOrigin.Z + b.LocalPos.Z),
			}
			for a := 0; a < 3; a++ {
				if first || p[a] < min[a] {
					min[a] = p[a]
				}
				if first || p[a]+1 > max[a] {
					max[a] = p[a] + 1
				}
			}
			first = false
		}
	}
	if first {
		return nil, fmt.Errorf("schematic: Sable blueprint contains no blocks")
	}
	sx, sy, sz := max[0]-min[0], max[1]-min[1], max[2]-min[2]
	if sx > MaxDimension || sy > MaxDimension || sz > MaxDimension {
		return nil, fmt.Errorf("schematic: Sable blueprint size %dx%dx%d exceeds maximum", sx, sy, sz)
	}
	if v := sx * sy * sz; v > MaxVolume {
		return nil, fmt.Errorf("schematic: Sable blueprint volume %d exceeds maximum", v)
	}

	s := New(sx, sy, sz)
	s.Meta.SourceFormat = "sable"
	s.Meta.LossyNotes = append(s.Meta.LossyNotes,
		"Sable blueprint flattened to blocks only: sub-level poses, previews and extra data are not carried over")
	if len(root.SubLevels) > 1 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf("%d sub-levels merged by block origin; relative poses (rotation/offset of moving parts) are ignored", len(root.SubLevels)))
	}

	beTotal := 0
	droppedEntities := 0
	unavailable := 0
	for _, sl := range root.SubLevels {
		if len(sl.BlockPalette) > MaxPaletteSize {
			return nil, fmt.Errorf("schematic: Sable sub-level palette exceeds maximum")
		}
		unavailableSet := map[int32]bool{}
		for _, id := range sl.UnavailablePaletteIDs {
			unavailableSet[id] = true
		}
		srcToModel := make([]int32, len(sl.BlockPalette))
		for i, e := range sl.BlockPalette {
			if e.Name == "" || unavailableSet[int32(i)] {
				srcToModel[i] = 0 // air
				unavailable++
				continue
			}
			if len(e.Name) > MaxBlockIDLength {
				return nil, fmt.Errorf("schematic: Sable palette entry %d invalid", i)
			}
			srcToModel[i] = s.PaletteIndex(BlockState{Name: e.Name, Properties: e.Properties})
		}

		beByID := sl.BlockEntities
		for _, b := range sl.Blocks {
			if b.PaletteID < 0 || int(b.PaletteID) >= len(srcToModel) {
				continue
			}
			x := int(sl.BlocksOrigin.X+b.LocalPos.X) - min[0]
			y := int(sl.BlocksOrigin.Y+b.LocalPos.Y) - min[1]
			z := int(sl.BlocksOrigin.Z+b.LocalPos.Z) - min[2]
			s.Blocks[s.Index(x, y, z)] = srcToModel[b.PaletteID]

			if b.BlockEntityDataID != nil {
				id := *b.BlockEntityDataID
				if id >= 0 && int(id) < len(beByID) {
					beTotal++
					if beTotal > MaxBlockEntities {
						return nil, fmt.Errorf("schematic: more than %d block entities", MaxBlockEntities)
					}
					fields, err := compoundFields(beByID[id])
					if err != nil {
						continue
					}
					delete(fields, "x")
					delete(fields, "y")
					delete(fields, "z")
					s.BlockEntities = append(s.BlockEntities, BlockEntity{
						Pos: [3]int{x, y, z},
						Raw: compoundFromFields(fields),
					})
				}
			}
		}
		droppedEntities += len(sl.Entities)
	}
	if unavailable > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf("%d palette entries were unavailable in the source and became air", unavailable))
	}
	if droppedEntities > 0 {
		s.Meta.LossyNotes = append(s.Meta.LossyNotes,
			fmt.Sprintf("%d entities (including Create contraptions) dropped", droppedEntities))
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}
