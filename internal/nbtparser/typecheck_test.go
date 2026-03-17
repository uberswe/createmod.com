package nbtparser

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"reflect"
	"testing"

	nbt "github.com/Tnze/go-mc/nbt"
)

// TestNBTRoundTripTypes verifies that decode→encode round-trip preserves
// semantic content. The Tnze encoder always writes []int32 as TAG_Int_Array
// and Go map iteration order is nondeterministic, so raw bytes won't match,
// but the decoded values must be identical.
func TestNBTRoundTripTypes(t *testing.T) {
	var origBuf bytes.Buffer
	encoder := nbt.NewEncoder(&origBuf)

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

	// Decode
	var decoded interface{}
	decoder := nbt.NewDecoder(bytes.NewReader(origBuf.Bytes()))
	rootTag, err := decoder.Decode(&decoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	t.Logf("Decoded root tag: %q", rootTag)
	t.Logf("Decoded type tree:")
	dumpInterfaceTypes(t, "  ", decoded)

	// Re-encode
	var reEncodedBuf bytes.Buffer
	encoder2 := nbt.NewEncoder(&reEncodedBuf)
	if err := encoder2.Encode(decoded, rootTag); err != nil {
		t.Fatalf("re-encode failed: %v", err)
	}

	t.Logf("Re-encoded bytes: %d", reEncodedBuf.Len())
	dumpTagTypes(t, "RE-ENCODED", reEncodedBuf.Bytes())

	// Verify semantic equivalence: decode re-encoded bytes and compare.
	// Raw bytes may differ due to Go map iteration order, but the decoded
	// values must be identical.
	var decoded2 interface{}
	dec2 := nbt.NewDecoder(bytes.NewReader(reEncodedBuf.Bytes()))
	if _, err := dec2.Decode(&decoded2); err != nil {
		t.Fatalf("re-decode failed: %v", err)
	}

	if !reflect.DeepEqual(decoded, decoded2) {
		t.Error("round-trip changed the decoded data")
	}
}

// TestRealSchematicRoundTrip verifies that decode→encode of a real schematic
// preserves semantic content. Raw bytes will differ because (a) Go map
// iteration order is nondeterministic and (b) the Tnze encoder converts
// TAG_List(TAG_Int) to TAG_Int_Array. The patchIntArrayToList workaround
// used by ReplacePalette restores the correct tag types; that path is tested
// separately in TestReplacePalettePreservesTagList.
func TestRealSchematicRoundTrip(t *testing.T) {
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
	decoder := nbt.NewDecoder(bytes.NewReader(decompressed))
	rootTag, err := decoder.Decode(&decoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Show decoded Go types for key fields
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

	// Verify semantic equivalence: decode re-encoded bytes and compare.
	// The Tnze decoder produces different Go types for TAG_List(TAG_Int)
	// ([]interface{}) vs TAG_Int_Array ([]int32), so we normalize both
	// before comparison.
	var decoded2 interface{}
	dec2 := nbt.NewDecoder(bytes.NewReader(reEncodedBuf.Bytes()))
	if _, err := dec2.Decode(&decoded2); err != nil {
		t.Fatalf("re-decode failed: %v", err)
	}

	norm1 := normalizeNBTValue(decoded)
	norm2 := normalizeNBTValue(decoded2)
	if !reflect.DeepEqual(norm1, norm2) {
		t.Error("round-trip changed the decoded data")
	}

	// Verify that patchIntArrayToList restores the original TAG_List types.
	listFields := findListIntFields(decompressed)
	if len(listFields) == 0 {
		t.Log("no TAG_List(TAG_Int) fields found in original — nothing to patch")
		return
	}
	t.Logf("TAG_List(TAG_Int) fields in original: %v", listFields)

	patched := patchIntArrayToList(reEncodedBuf.Bytes(), listFields)
	dumpTagTypes(t, "PATCHED", patched)

	// Confirm size and pos are TAG_List (0x09) after patching, not TAG_Int_Array (0x0B).
	for field := range listFields {
		assertFieldTagType(t, patched, field, 9, "patched output")
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

// normalizeNBTValue recursively converts NBT decoded values so that
// TAG_List(TAG_Int) (decoded as []interface{} of int32) and TAG_Int_Array
// (decoded as []int32) compare as equal.
func normalizeNBTValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, child := range val {
			out[k] = normalizeNBTValue(child)
		}
		return out
	case []interface{}:
		// Check if all elements are int32 — if so, normalize to []int32.
		allInt32 := len(val) > 0
		for _, elem := range val {
			if _, ok := elem.(int32); !ok {
				allInt32 = false
				break
			}
		}
		if allInt32 {
			ints := make([]int32, len(val))
			for i, elem := range val {
				ints[i] = elem.(int32)
			}
			return ints
		}
		// Otherwise recurse into each element.
		out := make([]interface{}, len(val))
		for i, elem := range val {
			out[i] = normalizeNBTValue(elem)
		}
		return out
	default:
		return v
	}
}

// assertFieldTagType checks that the first occurrence of the named field in raw
// NBT bytes has the expected tag type byte.
func assertFieldTagType(t *testing.T, data []byte, field string, wantTag byte, label string) {
	t.Helper()
	tagNames := map[byte]string{
		9: "TAG_List", 11: "TAG_Int_Array",
	}
	needle := []byte(field)
	idx := bytes.Index(data, needle)
	if idx < 3 {
		return
	}
	nameLen := int(data[idx-2])<<8 | int(data[idx-1])
	if nameLen != len(field) {
		return
	}
	gotTag := data[idx-3]
	if gotTag != wantTag {
		wantName := tagNames[wantTag]
		gotName := tagNames[gotTag]
		if wantName == "" {
			wantName = fmt.Sprintf("0x%02x", wantTag)
		}
		if gotName == "" {
			gotName = fmt.Sprintf("0x%02x", gotTag)
		}
		t.Errorf("[%s] field %q: got %s, want %s", label, field, gotName, wantName)
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
