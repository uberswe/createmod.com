package nbtparser

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"

	nbt "github.com/Tnze/go-mc/nbt"
)

// ReplaceBlock defines a single palette replacement.
type ReplaceBlock struct {
	OriginalID    string `json:"original"`    // e.g. "minecraft:oak_planks"
	ReplacementID string `json:"replacement"` // e.g. "minecraft:spruce_planks" or "" for removal
}

// blockFamilySuffix returns the family suffix of a block ID if it belongs to
// a known Minecraft block family (e.g. "_stairs", "_slab"). Returns "" if
// the block does not match any known family.
func blockFamilySuffix(blockID string) string {
	suffixes := []string{
		"_stairs", "_slab", "_planks", "_log", "_wood", "_wall", "_fence",
		"_fence_gate", "_door", "_trapdoor", "_button", "_pressure_plate",
		"_sign", "_hanging_sign", "_bricks", "_tiles",
	}
	// Strip namespace for suffix check
	parts := strings.SplitN(blockID, ":", 2)
	path := blockID
	if len(parts) == 2 {
		path = parts[1]
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(path, suffix) {
			return suffix
		}
	}
	return ""
}

// sameBlockFamily returns true if both block IDs share the same family suffix.
func sameBlockFamily(a, b string) bool {
	sa := blockFamilySuffix(a)
	sb := blockFamilySuffix(b)
	return sa != "" && sa == sb
}

// ReplacePalette applies block replacements to raw NBT data.
// It decodes the NBT, modifies matching palette entries, re-encodes,
// and re-compresses (gzip). Returns the modified NBT bytes.
func ReplacePalette(nbtData []byte, replacements []ReplaceBlock) ([]byte, error) {
	if len(replacements) == 0 {
		return nbtData, nil
	}

	// Validate all block IDs up front
	for _, r := range replacements {
		if !ValidateBlockID(r.OriginalID) {
			return nil, fmt.Errorf("invalid original block ID: %q", r.OriginalID)
		}
		if r.ReplacementID != "" && !ValidateBlockID(r.ReplacementID) {
			return nil, fmt.Errorf("invalid replacement block ID: %q", r.ReplacementID)
		}
	}

	// Build lookup map
	replMap := make(map[string]string, len(replacements))
	for _, r := range replacements {
		target := r.ReplacementID
		if target == "" {
			target = "minecraft:air"
		}
		replMap[r.OriginalID] = target
	}

	// Decompress
	decompressed, err := decompressLimited(nbtData)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	// Decode NBT into generic interface
	var rootTag string
	var data interface{}
	decoder := nbt.NewDecoder(bytes.NewReader(decompressed))
	rootTag, err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("NBT decode failed: %w", err)
	}

	// Navigate to palette array
	rootMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("NBT root is not a compound tag")
	}

	paletteRaw, exists := rootMap["palette"]
	if !exists {
		return nil, fmt.Errorf("no palette field found in NBT data")
	}

	paletteSlice, ok := paletteRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("palette is not an array")
	}

	// Apply replacements to palette entries
	for i, entry := range paletteSlice {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		nameRaw, exists := entryMap["Name"]
		if !exists {
			continue
		}
		name, ok := nameRaw.(string)
		if !ok {
			continue
		}

		replacement, found := replMap[name]
		if !found {
			continue
		}

		if replacement == "minecraft:air" {
			// Removal: set to air and clear properties
			entryMap["Name"] = replacement
			delete(entryMap, "Properties")
		} else {
			// Replacement: change name, handle properties
			if sameBlockFamily(name, replacement) {
				// Same family: keep existing properties
				entryMap["Name"] = replacement
			} else {
				// Different family: strip properties to avoid invalid states
				entryMap["Name"] = replacement
				delete(entryMap, "Properties")
			}
		}
		paletteSlice[i] = entryMap
	}

	rootMap["palette"] = paletteSlice

	// Re-encode NBT
	var nbtBuf bytes.Buffer
	encoder := nbt.NewEncoder(&nbtBuf)
	if err := encoder.Encode(data, rootTag); err != nil {
		return nil, fmt.Errorf("NBT encode failed: %w", err)
	}

	// Re-compress with gzip (Create schematics are always gzip)
	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)
	if _, err := gzw.Write(nbtBuf.Bytes()); err != nil {
		return nil, fmt.Errorf("gzip write failed: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	return gzBuf.Bytes(), nil
}
