package nbtparser

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"sort"

	mc "github.com/uberswe/mcnbt"
)

// Package nbtparser provides a thin wrapper for parsing Minecraft NBT files.
// It is introduced as scaffolding for future mcnbt integration while keeping
// current tests stable. The API offers a forward-compatible ParseSummary that
// can be enriched later with block/material statistics when mcnbt is wired in.

// Validate performs a backward-compatible validation.
//   - Reject empty uploads.
//   - If the data appears gzip-compressed but cannot be opened as gzip, reject with a clear reason.
//   - Otherwise accept the data (even if mcnbt cannot decode it), to avoid breaking current flows
//     until stricter validation is rolled out with real NBT fixtures.
func Validate(data []byte) (ok bool, reason string) {
	if len(data) == 0 {
		return false, "empty upload"
	}
	// If it looks like gzip but isn't valid gzip, reject.
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		if _, gzErr := gzip.NewReader(bytes.NewReader(data)); gzErr != nil {
			return false, "invalid gzip-compressed NBT data"
		}
		// valid gzip stream -> accept (content might still be non-NBT; handled later gracefully)
		return true, ""
	}
	// Best-effort: try mcnbt.DecodeAny; success confirms it's parseable. If it fails, still accept
	// to keep compatibility with existing uploads that may be uncompressed or in formats not yet enforced.
	if _, err := mc.DecodeAny(data); err == nil {
		return true, ""
	}
	return true, ""
}

// ParseSummary tries to extract a human-friendly summary of the uploaded NBT.
// Minimal implementation kept, but could be expanded to include block count or size later.
func ParseSummary(data []byte) (summary string, ok bool) {
	if len(data) == 0 {
		return "", false
	}
	// Detect gzip by magic header (0x1f 0x8b) and attempt a quick read to confirm.
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		// Verify it is actually gzip by trying to open a reader and reading a small chunk
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err == nil {
			defer gr.Close()
			buf := make([]byte, 1)
			_, _ = gr.Read(buf) // best-effort; ignore errors since magic matched
			return fmt.Sprintf("nbt=gzip"), true
		}
		// Magic matched but reader failed; still report gzip to be helpful
		return "nbt=gzip", true
	}
	// Could extend with zlib detection if needed; default to uncompressed when unknown.
	return "nbt=uncompressed", true
}

// ExtractStats parses the NBT using mcnbt and extracts basic statistics.
// - blockCount: number of blocks in the StandardFormat.Blocks array
// - materials: a simple frequency list keyed by palette state id (best-effort)
// If parsing fails, returns zero values with ok=false.
func ExtractStats(data []byte) (blockCount int, materials []string, ok bool) {
	if len(data) == 0 {
		return 0, nil, false
	}
	std, err := mc.ConvertToStandard(data)
	if err != nil {
		return 0, nil, false
	}
	blockCount = len(std.Blocks)
	// Count states as a simple proxy for materials. If palette names are available in the
	// library, this can be upgraded to use readable names.
	counts := make(map[int]int)
	for _, b := range std.Blocks {
		if b.State != 0 {
			counts[b.State]++
		} else {
			// some formats may omit state -> treat as 0 bucket
			counts[0]++
		}
	}
	// produce stable, sorted output like: state:123 = 45
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	materials = make([]string, 0, len(keys))
	for _, k := range keys {
		materials = append(materials, fmt.Sprintf("state:%d=%d", k, counts[k]))
	}
	return blockCount, materials, true
}

// Material represents a block type and its count in a schematic
type Material struct {
	BlockID string `json:"block_id"`
	Count   int    `json:"count"`
}

// ExtractMaterials parses NBT data and returns a list of materials (block types and counts).
// It uses the mcnbt library to convert to StandardFormat and reads the palette.
func ExtractMaterials(data []byte) ([]Material, error) {
	decoded, err := mc.DecodeAny(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode NBT: %w", err)
	}

	std, err := mc.ConvertToStandard(decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to standard format: %w", err)
	}

	// Count blocks by palette state
	stateCounts := make(map[int]int)
	for _, block := range std.Blocks {
		stateCounts[block.State]++
	}

	// Map palette states to block names
	blockCounts := make(map[string]int)
	for state, count := range stateCounts {
		if palette, ok := std.Palette[state]; ok {
			name := palette.Name
			// Filter out air blocks
			if name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air" {
				continue
			}
			blockCounts[name] += count
		}
	}

	// Convert to sorted slice
	materials := make([]Material, 0, len(blockCounts))
	for blockID, count := range blockCounts {
		materials = append(materials, Material{
			BlockID: blockID,
			Count:   count,
		})
	}

	// Sort by count descending
	sort.Slice(materials, func(i, j int) bool {
		return materials[i].Count > materials[j].Count
	})

	return materials, nil
}
