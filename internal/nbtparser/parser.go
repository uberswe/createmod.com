package nbtparser

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"sort"

	mc "github.com/uberswe/mcnbt"
)

// Package nbtparser provides a thin wrapper for parsing Minecraft NBT files.
// It is introduced as scaffolding for future mcnbt integration while keeping
// current tests stable. The API offers a forward-compatible ParseSummary that
// can be enriched later with block/material statistics when mcnbt is wired in.

// maxDecompressedSize is the maximum allowed size after decompression (100 MB).
// This prevents decompression bombs where a small gzip expands to gigabytes.
const maxDecompressedSize = 100 * 1024 * 1024

// maxBlockIDLength is the maximum allowed length for a Minecraft block ID.
const maxBlockIDLength = 256

// maxDimension is the upper bound for schematic dimensions.
// Minecraft structure blocks support up to 48x48x48, but Create mod
// schematics can be much larger. 32768 accommodates very large builds.
const maxDimension = 32768

// blockIDPattern matches valid Minecraft resource locations: namespace:path
// where both parts consist of [a-z0-9_.-] and path may contain /.
var blockIDPattern = regexp.MustCompile(`^[a-z0-9_.\-]+:[a-z0-9_.\-/]+$`)

// ValidateBlockID checks whether a block ID conforms to Minecraft's
// resource location format (e.g. "minecraft:stone").
func ValidateBlockID(id string) bool {
	if len(id) == 0 || len(id) > maxBlockIDLength {
		return false
	}
	return blockIDPattern.MatchString(id)
}

// decompressLimited decompresses gzip or zlib data with a size cap.
// Returns the raw bytes if the data is not compressed or after
// successful decompression. Returns an error if the decompressed
// data exceeds maxDecompressedSize.
func decompressLimited(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	var r io.Reader
	compressed := false

	if len(data) >= 2 {
		if data[0] == 0x1f && data[1] == 0x8b {
			// Gzip magic number
			gr, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("invalid gzip data: %w", err)
			}
			defer gr.Close()
			r = gr
			compressed = true
		} else if data[0] == 0x78 && (data[1] == 0x01 || data[1] == 0x9c || data[1] == 0xda) {
			// Zlib magic number
			zr, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("invalid zlib data: %w", err)
			}
			defer zr.Close()
			r = zr
			compressed = true
		}
	}

	if !compressed {
		// Not compressed — the raw data is already bounded by the upload
		// size limit (10 MB), so no additional cap needed.
		return data, nil
	}

	// Read decompressed data with a hard limit to prevent decompression bombs.
	limited := io.LimitReader(r, maxDecompressedSize+1)
	decompressed, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}
	if len(decompressed) > maxDecompressedSize {
		return nil, fmt.Errorf("decompressed data exceeds %d byte limit", maxDecompressedSize)
	}
	return decompressed, nil
}

// Validate performs a backward-compatible validation.
//   - Reject empty uploads.
//   - If the data appears gzip-compressed but cannot be opened as gzip, reject with a clear reason.
//   - Reject files that decompress beyond the size limit.
//   - Otherwise accept the data (even if mcnbt cannot decode it), to avoid breaking current flows
//     until stricter validation is rolled out with real NBT fixtures.
func Validate(data []byte) (ok bool, reason string) {
	if len(data) == 0 {
		return false, "empty upload"
	}

	_, err := decompressLimited(data)
	if err != nil {
		return false, err.Error()
	}

	return true, ""
}

// ParseSummary tries to extract a human-friendly summary of the uploaded NBT.
// Minimal implementation kept, but could be expanded to include block count or size later.
func ParseSummary(data []byte) (summary string, ok bool) {
	if len(data) == 0 {
		return "", false
	}
	// Detect gzip by magic header (0x1f 0x8b).
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return "nbt=gzip", true
	}
	// Could extend with zlib detection if needed; default to uncompressed when unknown.
	return "nbt=uncompressed", true
}

// clampDimension restricts a dimension value to [0, maxDimension].
func clampDimension(v int) int {
	if v < 0 {
		return 0
	}
	if v > maxDimension {
		return maxDimension
	}
	return v
}

// ExtractStats parses the NBT using mcnbt and extracts basic statistics.
// - blockCount: number of blocks in the StandardFormat.Blocks array
// - materials: a simple frequency list keyed by palette state id (best-effort)
// If parsing fails via ConvertToStandard, falls back to raw map extraction.
func ExtractStats(data []byte) (blockCount int, materials []string, ok bool) {
	if len(data) == 0 {
		return 0, nil, false
	}

	safe, err := decompressLimited(data)
	if err != nil {
		return 0, nil, false
	}

	// Try the standard path first
	std, err := mc.ConvertToStandard(safe)
	if err != nil {
		// Fallback: try decoding and extracting from raw map
		decoded, decErr := mc.DecodeAny(safe)
		if decErr != nil {
			return 0, nil, false
		}
		return extractStatsFromMap(decoded)
	}

	blockCount = len(std.Blocks)
	counts := make(map[int]int)
	for _, b := range std.Blocks {
		if b.State != 0 {
			counts[b.State]++
		} else {
			counts[0]++
		}
	}
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

// ExtractDimensions parses NBT data and returns the schematic dimensions.
// Dimensions are clamped to [0, maxDimension] to prevent unreasonable values.
// Falls back to raw map extraction if ConvertToStandard fails.
func ExtractDimensions(data []byte) (x, y, z int, ok bool) {
	safe, err := decompressLimited(data)
	if err != nil {
		return 0, 0, 0, false
	}

	decoded, err := mc.DecodeAny(safe)
	if err != nil {
		return 0, 0, 0, false
	}
	std, err := mc.ConvertToStandard(decoded)
	if err == nil {
		return clampDimension(std.Size.X), clampDimension(std.Size.Y), clampDimension(std.Size.Z), true
	}
	// Fallback: extract from raw map
	rx, ry, rz, rok := extractDimensionsFromMap(decoded)
	if !rok {
		return 0, 0, 0, false
	}
	return clampDimension(rx), clampDimension(ry), clampDimension(rz), true
}

// Material represents a block type and its count in a schematic
type Material struct {
	BlockID string `json:"block_id"`
	Count   int    `json:"count"`
}

// SanitizeMaterials filters a materials list, removing entries with
// invalid block IDs. This should be called before storing or displaying
// materials extracted from untrusted NBT data.
func SanitizeMaterials(mats []Material) []Material {
	result := make([]Material, 0, len(mats))
	for _, m := range mats {
		if ValidateBlockID(m.BlockID) {
			result = append(result, m)
		}
	}
	return result
}

// ExtractMaterials parses NBT data and returns a list of materials (block types and counts).
// Invalid block IDs are filtered out. Falls back to raw map extraction if
// ConvertToStandard fails.
func ExtractMaterials(data []byte) ([]Material, error) {
	safe, err := decompressLimited(data)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	decoded, err := mc.DecodeAny(safe)
	if err != nil {
		return nil, fmt.Errorf("failed to decode NBT: %w", err)
	}

	std, err := mc.ConvertToStandard(decoded)
	if err != nil {
		// Fallback: extract from raw map
		mats, fallbackErr := extractMaterialsFromMap(decoded)
		if fallbackErr != nil {
			return nil, fmt.Errorf("standard conversion failed: %v; fallback also failed: %w", err, fallbackErr)
		}
		return SanitizeMaterials(mats), nil
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

	return SanitizeMaterials(materials), nil
}

// --- Fallback extraction from raw decoded map ---
// These functions handle the case where mcnbt.ConvertToStandard fails
// (e.g. due to type mismatches in entity fields) by reading directly
// from the decoded map[string]interface{} structure.

// derefMap dereferences *interface{} and returns the underlying map if present.
func derefMap(v interface{}) map[string]interface{} {
	if ptr, ok := v.(*interface{}); ok && ptr != nil {
		return derefMap(*ptr)
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// extractDimensionsFromMap reads the "size" field from a Create-format NBT map.
// Create format stores size as a list of 3 ints: [x, y, z].
func extractDimensionsFromMap(decoded interface{}) (x, y, z int, ok bool) {
	m := derefMap(decoded)
	if m == nil {
		return 0, 0, 0, false
	}
	sizeVal, exists := m["size"]
	if !exists {
		return 0, 0, 0, false
	}
	sizeSlice, ok := toIntSlice(sizeVal)
	if !ok || len(sizeSlice) < 3 {
		return 0, 0, 0, false
	}
	return sizeSlice[0], sizeSlice[1], sizeSlice[2], true
}

// extractMaterialsFromMap reads "blocks" and "palette" from a Create-format NBT map.
func extractMaterialsFromMap(decoded interface{}) ([]Material, error) {
	m := derefMap(decoded)
	if m == nil {
		return nil, fmt.Errorf("decoded data is not a map")
	}

	// Read palette: []map with "Name" key
	paletteRaw, ok := m["palette"]
	if !ok {
		return nil, fmt.Errorf("no palette field found")
	}
	paletteSlice, ok := paletteRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("palette is not an array")
	}
	paletteNames := make([]string, len(paletteSlice))
	for i, entry := range paletteSlice {
		if em, ok := entry.(map[string]interface{}); ok {
			if name, ok := em["Name"].(string); ok {
				paletteNames[i] = name
			}
		}
	}

	// Read blocks: []map with "state" and optionally "pos"
	blocksRaw, ok := m["blocks"]
	if !ok {
		return nil, fmt.Errorf("no blocks field found")
	}
	blocksSlice, ok := blocksRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("blocks is not an array")
	}

	// Count blocks per state
	stateCounts := make(map[int]int)
	for _, block := range blocksSlice {
		bm, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		state := toInt(bm["state"])
		stateCounts[state]++
	}

	// Map states to block names and aggregate
	blockCounts := make(map[string]int)
	for state, count := range stateCounts {
		name := ""
		if state >= 0 && state < len(paletteNames) {
			name = paletteNames[state]
		}
		if name == "" {
			name = fmt.Sprintf("unknown:%d", state)
		}
		// Filter air
		if name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air" {
			continue
		}
		blockCounts[name] += count
	}

	materials := make([]Material, 0, len(blockCounts))
	for blockID, count := range blockCounts {
		materials = append(materials, Material{
			BlockID: blockID,
			Count:   count,
		})
	}
	sort.Slice(materials, func(i, j int) bool {
		return materials[i].Count > materials[j].Count
	})
	return materials, nil
}

// extractStatsFromMap is the fallback for ExtractStats using raw map data.
func extractStatsFromMap(decoded interface{}) (blockCount int, materials []string, ok bool) {
	m := derefMap(decoded)
	if m == nil {
		return 0, nil, false
	}
	blocksRaw, exists := m["blocks"]
	if !exists {
		return 0, nil, false
	}
	blocksSlice, isSlice := blocksRaw.([]interface{})
	if !isSlice {
		return 0, nil, false
	}
	blockCount = len(blocksSlice)

	stateCounts := make(map[int]int)
	for _, block := range blocksSlice {
		bm, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		state := toInt(bm["state"])
		stateCounts[state]++
	}
	keys := make([]int, 0, len(stateCounts))
	for k := range stateCounts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	materials = make([]string, 0, len(keys))
	for _, k := range keys {
		materials = append(materials, fmt.Sprintf("state:%d=%d", k, stateCounts[k]))
	}
	return blockCount, materials, true
}

// toInt converts a numeric interface{} value to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// toIntSlice converts a slice interface{} to []int.
func toIntSlice(v interface{}) ([]int, bool) {
	switch s := v.(type) {
	case []interface{}:
		result := make([]int, len(s))
		for i, val := range s {
			result[i] = toInt(val)
		}
		return result, true
	case []int32:
		result := make([]int, len(s))
		for i, val := range s {
			result[i] = int(val)
		}
		return result, true
	case []int64:
		result := make([]int, len(s))
		for i, val := range s {
			result[i] = int(val)
		}
		return result, true
	case []int:
		return s, true
	default:
		return nil, false
	}
}
