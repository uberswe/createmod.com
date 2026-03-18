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

	// Record which fields use TAG_List(TAG_Int) in the original data so we
	// can restore them after re-encoding. The Tnze encoder always converts
	// int slices to TAG_Int_Array, but Create mod requires TAG_List for
	// fields like "size" and block "pos".
	listFields := findListIntFields(decompressed)

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

	// Patch TAG_Int_Array back to TAG_List(TAG_Int) for fields that were
	// originally TAG_List. The Tnze encoder always writes TAG_Int_Array for
	// int slices, but Create mod requires TAG_List for "size" and "pos".
	nbtBytes := patchIntArrayToList(nbtBuf.Bytes(), listFields)

	// Re-compress with gzip (Create schematics are always gzip)
	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)
	if _, err := gzw.Write(nbtBytes); err != nil {
		return nil, fmt.Errorf("gzip write failed: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	return gzBuf.Bytes(), nil
}

// findListIntFields scans raw NBT bytes and returns the set of field names
// that are encoded as TAG_List with TAG_Int elements. These fields must be
// preserved as TAG_List after re-encoding because Create mod does not accept
// TAG_Int_Array for fields like "size" and block "pos".
func findListIntFields(data []byte) map[string]bool {
	fields := make(map[string]bool)
	for i := 0; i < len(data)-3; i++ {
		if data[i] != 9 { // TAG_List = 9
			continue
		}
		nameLen := int(data[i+1])<<8 | int(data[i+2])
		if nameLen <= 0 || i+3+nameLen >= len(data) {
			continue
		}
		name := string(data[i+3 : i+3+nameLen])
		elemTypeIdx := i + 3 + nameLen
		if elemTypeIdx < len(data) && data[elemTypeIdx] == 3 { // TAG_Int = 3
			fields[name] = true
		}
	}
	return fields
}

// patchIntArrayToList converts TAG_Int_Array entries back to TAG_List(TAG_Int)
// for the specified field names. The Tnze/go-mc/nbt encoder always writes int
// slices as TAG_Int_Array (tag 0x0B), but Create mod schematics require
// TAG_List (tag 0x09) for fields like "size" and "pos".
//
// TAG_Int_Array format: 0x0B + nameLen(2) + name(n) + count(4) + values(count*4)
// TAG_List format:      0x09 + nameLen(2) + name(n) + elemType(1) + count(4) + values(count*4)
//
// The patch changes the tag type byte and inserts one element-type byte (0x03)
// after the field name.
func patchIntArrayToList(data []byte, listFields map[string]bool) []byte {
	if len(listFields) == 0 {
		return data
	}

	// Collect patch positions
	type patchInfo struct {
		tagTypePos int
		nameLen    int
	}
	var patches []patchInfo

	for i := 0; i < len(data)-3; i++ {
		if data[i] != 0x0B { // TAG_Int_Array
			continue
		}
		nameLen := int(data[i+1])<<8 | int(data[i+2])
		if nameLen <= 0 || i+3+nameLen > len(data) {
			continue
		}
		name := string(data[i+3 : i+3+nameLen])
		if listFields[name] {
			patches = append(patches, patchInfo{i, nameLen})
		}
	}

	if len(patches) == 0 {
		return data
	}

	// Rebuild with patches: each patch inserts 1 byte (element type)
	result := make([]byte, 0, len(data)+len(patches))
	prev := 0
	for _, p := range patches {
		// Copy everything up to and including the tag type byte position
		result = append(result, data[prev:p.tagTypePos]...)
		// Write TAG_List type byte instead of TAG_Int_Array
		result = append(result, 0x09)
		// Copy name length + name bytes
		nameEnd := p.tagTypePos + 3 + p.nameLen
		result = append(result, data[p.tagTypePos+1:nameEnd]...)
		// Insert element type byte: TAG_Int = 3
		result = append(result, 0x03)
		prev = nameEnd
	}
	result = append(result, data[prev:]...)

	return result
}
