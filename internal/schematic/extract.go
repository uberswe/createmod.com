package schematic

import "sort"

// MaterialCount is a block id with its total count, sorted descending.
type MaterialCount struct {
	BlockID string
	Count   int
}

// Materials returns the material list (non-air blocks grouped by block id,
// ignoring properties), sorted by count descending then id.
func (s *Schematic) Materials() []MaterialCount {
	counts := make(map[string]int)
	perPalette := make([]int, len(s.Palette))
	for _, idx := range s.Blocks {
		perPalette[idx]++
	}
	for i, st := range s.Palette {
		if st.IsAir() || perPalette[i] == 0 {
			continue
		}
		counts[st.Name] += perPalette[i]
	}
	out := make([]MaterialCount, 0, len(counts))
	for id, c := range counts {
		out = append(out, MaterialCount{BlockID: id, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].BlockID < out[j].BlockID
	})
	return out
}

// BlockCount returns the number of non-air blocks.
func (s *Schematic) BlockCount() int {
	perPalette := make([]int, len(s.Palette))
	for _, idx := range s.Blocks {
		perPalette[idx]++
	}
	total := 0
	for i, st := range s.Palette {
		if !st.IsAir() {
			total += perPalette[i]
		}
	}
	return total
}

// Caps describes what a given build supports; it drives the download
// component's per-format menu and the world-export size guard.
type Caps struct {
	Size             [3]int
	Volume           int
	BlockCount       int
	HasBlockEntities bool
	HasEntities      bool
	DataVersion      int // 0 = unknown
	// WorldExportable is false when the build exceeds the world-export
	// volume guard (generation runs in the request path).
	WorldExportable bool
}

// MaxWorldExportVolume caps on-demand world export; larger builds must be
// downloaded as schematics instead.
const MaxWorldExportVolume = 16 * 1024 * 1024

// Capabilities computes the capability summary for the model.
func (s *Schematic) Capabilities() Caps {
	v := s.Volume()
	return Caps{
		Size:             s.Size,
		Volume:           v,
		BlockCount:       s.BlockCount(),
		HasBlockEntities: len(s.BlockEntities) > 0,
		HasEntities:      len(s.Entities) > 0,
		DataVersion:      s.DataVersion,
		WorldExportable:  v <= MaxWorldExportVolume,
	}
}
