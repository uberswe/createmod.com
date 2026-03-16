package nbtparser

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"testing"

	nbt "github.com/Tnze/go-mc/nbt"
)

// TestNBTRoundTripTypes checks whether Tnze/go-mc/nbt preserves
// tag types during decode→encode round-trip. Create mod schematics use
// TAG_List(TAG_Int) for "size" and block "pos" fields. If the encoder
// converts these to TAG_Int_Array, Create mod won't load the schematic.
func TestNBTRoundTripTypes(t *testing.T) {
	// Build a minimal Create-format schematic with TAG_List for size and pos
	var origBuf bytes.Buffer
	encoder := nbt.NewEncoder(&origBuf)

	// Manually build a structure similar to Create schematics
	data := map[string]interface{}{
		"size": []int32{int32(10), int32(20), int32(30)},
		"palette": []interface{}{
			map[string]interface{}{
				"Name": "minecraft:stone",
			},
		},
		"blocks": []interface{}{
			map[string]interface{}{
				"state": int32(0),
				"pos":   []int32{int32(1), int32(2), int32(3)},
			},
		},
	}

	if err := encoder.Encode(data, ""); err != nil {
		t.Fatalf("initial encode failed: %v", err)
	}

	t.Logf("Original encoded bytes: %d", origBuf.Len())
	dumpTagTypes(t, "ORIGINAL", origBuf.Bytes())

	// Now decode and re-encode (same as ReplacePalette does)
	var decoded interface{}
	var rootTag string
	decoder := nbt.NewDecoder(bytes.NewReader(origBuf.Bytes()))
	rootTag, err := decoder.Decode(&decoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	t.Logf("Decoded root tag: %q", rootTag)
	t.Logf("Decoded type tree:")
	dumpInterfaceTypes(t, "  ", decoded)

	var reEncodedBuf bytes.Buffer
	encoder2 := nbt.NewEncoder(&reEncodedBuf)
	if err := encoder2.Encode(decoded, rootTag); err != nil {
		t.Fatalf("re-encode failed: %v", err)
	}

	t.Logf("Re-encoded bytes: %d", reEncodedBuf.Len())
	dumpTagTypes(t, "RE-ENCODED", reEncodedBuf.Bytes())

	// Compare
	if !bytes.Equal(origBuf.Bytes(), reEncodedBuf.Bytes()) {
		t.Error("Round-trip changed the NBT bytes! Likely tag type mismatch.")
		t.Logf("Original hex (first 100): %x", truncBytes(origBuf.Bytes(), 100))
		t.Logf("Re-encoded hex (first 100): %x", truncBytes(reEncodedBuf.Bytes(), 100))
	} else {
		t.Log("Round-trip preserved bytes exactly")
	}
}

// TestRealSchematicRoundTrip tests with the actual modified schematic file
func TestRealSchematicRoundTrip(t *testing.T) {
	// Compare original vs modified
	origPath := "../../large_warehouse_b3x6lmwzw4.nbt"
	modPath := "../../large-warehouse-modified.nbt"
	origData, err1 := readTestFile(origPath)
	modData, err2 := readTestFile(modPath)
	if err1 == nil && err2 == nil {
		origDecomp, _ := decompressLimited(origData)
		modDecomp, _ := decompressLimited(modData)
		t.Log("=== ORIGINAL FILE ===")
		dumpTagTypes(t, "ORIGINAL-FILE", origDecomp)
		t.Log("=== MODIFIED FILE ===")
		dumpTagTypes(t, "MODIFIED-FILE", modDecomp)
	}

	const path = "../../large_warehouse_b3x6lmwzw4.nbt"
	data, err := readTestFile(path)
	if err != nil {
		t.Skipf("skipping: %v", err)
	}

	decompressed, err := decompressLimited(data)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	t.Log("=== BEFORE round-trip ===")
	dumpTagTypes(t, "ORIGINAL", decompressed)

	// Decode
	var decoded interface{}
	var rootTag string
	decoder := nbt.NewDecoder(bytes.NewReader(decompressed))
	rootTag, err = decoder.Decode(&decoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Show decoded Go types
	if m, ok := decoded.(map[string]interface{}); ok {
		if sizeVal, exists := m["size"]; exists {
			t.Logf("Decoded 'size' Go type: %T value: %v", sizeVal, sizeVal)
		}
		if blocksVal, exists := m["blocks"]; exists {
			if blocks, ok := blocksVal.([]interface{}); ok && len(blocks) > 0 {
				if block, ok := blocks[0].(map[string]interface{}); ok {
					if posVal, exists := block["pos"]; exists {
						t.Logf("Decoded 'blocks[0].pos' Go type: %T value: %v", posVal, posVal)
					}
				}
			}
		}
	}

	// Re-encode
	var reEncodedBuf bytes.Buffer
	encoder := nbt.NewEncoder(&reEncodedBuf)
	if err := encoder.Encode(decoded, rootTag); err != nil {
		t.Fatalf("re-encode failed: %v", err)
	}

	t.Log("=== AFTER round-trip ===")
	dumpTagTypes(t, "RE-ENCODED", reEncodedBuf.Bytes())

	if !bytes.Equal(decompressed, reEncodedBuf.Bytes()) {
		t.Error("Round-trip changed the NBT bytes!")
	}
}

func truncBytes(b []byte, n int) []byte {
	if len(b) > n {
		return b[:n]
	}
	return b
}

func readTestFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// dumpTagTypes reads raw NBT bytes and logs the tag types found for key fields
func dumpTagTypes(t *testing.T, label string, data []byte) {
	t.Helper()
	// NBT tag type IDs
	tagNames := map[byte]string{
		0: "TAG_End", 1: "TAG_Byte", 2: "TAG_Short", 3: "TAG_Int",
		4: "TAG_Long", 5: "TAG_Float", 6: "TAG_Double", 7: "TAG_Byte_Array",
		8: "TAG_String", 9: "TAG_List", 10: "TAG_Compound", 11: "TAG_Int_Array",
		12: "TAG_Long_Array",
	}

	// Search for "size" and "pos" field names in the raw bytes and report the tag type byte
	searchFields := []string{"size", "pos", "blocks", "palette"}
	for _, field := range searchFields {
		needle := []byte(field)
		idx := 0
		for {
			pos := bytes.Index(data[idx:], needle)
			if pos < 0 {
				break
			}
			absPos := idx + pos
			// The tag type byte is 1 byte before the name length (2 bytes before name)
			// Format: TagType(1) + NameLength(2) + Name(n)
			nameStart := absPos
			if nameStart >= 3 {
				nameLen := int(data[nameStart-2])<<8 | int(data[nameStart-1])
				if nameLen == len(field) {
					tagType := data[nameStart-3]
					typeName := tagNames[tagType]
					if typeName == "" {
						typeName = fmt.Sprintf("UNKNOWN(%d)", tagType)
					}
					// If it's a TAG_List, the next byte after the name is the element type
					extra := ""
					if tagType == 9 {
						elemIdx := nameStart + len(field)
						if elemIdx < len(data) {
							elemType := data[elemIdx]
							elemName := tagNames[elemType]
							if elemName == "" {
								elemName = fmt.Sprintf("UNKNOWN(%d)", elemType)
							}
							extra = fmt.Sprintf(" [element_type=%s]", elemName)
						}
					}
					t.Logf("[%s] field %q at byte %d: %s%s", label, field, nameStart-3, typeName, extra)
				}
			}
			idx = absPos + len(needle)
		}
	}
}

func dumpInterfaceTypes(t *testing.T, indent string, v interface{}) {
	t.Helper()
	switch val := v.(type) {
	case map[string]interface{}:
		for k, child := range val {
			t.Logf("%s%q: %T", indent, k, child)
			if k == "size" || k == "pos" {
				dumpInterfaceTypes(t, indent+"  ", child)
			}
		}
	case []interface{}:
		if len(val) > 0 {
			t.Logf("%s[0]: %T", indent, val[0])
		}
	case []int32:
		t.Logf("%svalues: %v", indent, val)
	case []int64:
		t.Logf("%svalues: %v", indent, val)
	default:
		t.Logf("%svalue: %v", indent, val)
	}
}

// TestGzipRoundTrip tests the full gzip compress/decompress path
func TestGzipRoundTrip(t *testing.T) {
	// Create minimal NBT with a TAG_List of TAG_Int for "size"
	var origBuf bytes.Buffer
	encoder := nbt.NewEncoder(&origBuf)
	data := map[string]interface{}{
		"size": []int32{int32(10), int32(20), int32(30)},
	}
	if err := encoder.Encode(data, ""); err != nil {
		t.Fatal(err)
	}

	// Compress
	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)
	gzw.Write(origBuf.Bytes())
	gzw.Close()

	// Run through ReplacePalette with no replacements (should be identity)
	result, err := ReplacePalette(gzBuf.Bytes(), []ReplaceBlock{})
	if err != nil {
		t.Fatalf("ReplacePalette with no replacements failed: %v", err)
	}

	// Decompress result
	gr, _ := gzip.NewReader(bytes.NewReader(result))
	var resultBuf bytes.Buffer
	resultBuf.ReadFrom(gr)

	if !bytes.Equal(origBuf.Bytes(), resultBuf.Bytes()) {
		t.Log("=== ORIGINAL ===")
		dumpTagTypes(t, "ORIG", origBuf.Bytes())
		t.Log("=== AFTER REPLACE_PALETTE ===")
		dumpTagTypes(t, "RESULT", resultBuf.Bytes())
		t.Error("ReplacePalette with empty replacements changed the NBT!")
	}
}

// TestReplacePalettePreservesTagList verifies that ReplacePalette outputs
// TAG_List(TAG_Int) for size and pos, not TAG_Int_Array.
func TestReplacePalettePreservesTagList(t *testing.T) {
	const path = "../../large_warehouse_b3x6lmwzw4.nbt"
	data, err := readTestFile(path)
	if err != nil {
		t.Skipf("skipping: %v", err)
	}

	// Apply a replacement through ReplacePalette
	modified, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:stone", ReplacementID: "minecraft:diamond_block"},
	})
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	modDecomp, err := decompressLimited(modified)
	if err != nil {
		t.Fatalf("decompress modified failed: %v", err)
	}

	// Check that size and pos are TAG_List, not TAG_Int_Array
	tagNames := map[byte]string{
		9: "TAG_List", 11: "TAG_Int_Array",
	}
	for _, field := range []string{"size", "pos"} {
		needle := []byte(field)
		idx := bytes.Index(modDecomp, needle)
		if idx < 3 {
			continue
		}
		nameLen := int(modDecomp[idx-2])<<8 | int(modDecomp[idx-1])
		if nameLen != len(field) {
			continue
		}
		tagType := modDecomp[idx-3]
		typeName := tagNames[tagType]
		if tagType == 11 {
			t.Errorf("field %q is %s (TAG_Int_Array) — Create mod requires TAG_List(TAG_Int)", field, typeName)
		} else if tagType == 9 {
			t.Logf("field %q is correctly TAG_List", field)
		} else {
			t.Logf("field %q has tag type %d", field, tagType)
		}
	}
}
