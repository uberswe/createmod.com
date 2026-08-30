package nbtparser

import (
	"bytes"
	"compress/gzip"
	"testing"

	nbt "github.com/Tnze/go-mc/nbt"
)

// buildStructureWithBE builds a minimal 1x1x2 vanilla-structure NBT where the
// second block is a Create redstone link carrying block-entity data
// ("Frequency" + "id"), gzip-compressed like a real .nbt upload.
func buildStructureWithBE(t *testing.T) []byte {
	t.Helper()
	root := map[string]interface{}{
		"size": []int32{1, 1, 2},
		"palette": []interface{}{
			map[string]interface{}{"Name": "minecraft:stone"},
			map[string]interface{}{"Name": "create:redstone_link"},
		},
		"blocks": []interface{}{
			map[string]interface{}{"pos": []int32{0, 0, 0}, "state": int32(0)},
			map[string]interface{}{
				"pos":   []int32{0, 0, 1},
				"state": int32(1),
				"nbt": map[string]interface{}{
					"id": "create:redstone_link",
					"Frequency": []interface{}{
						map[string]interface{}{"Value": map[string]interface{}{"id": "minecraft:redstone", "Count": int8(1)}},
						map[string]interface{}{"Value": map[string]interface{}{"id": "minecraft:iron_ingot", "Count": int8(1)}},
					},
					"Receiver": int8(1),
				},
			},
		},
	}
	var plain bytes.Buffer
	if err := nbt.NewEncoder(&plain).Encode(root, ""); err != nil {
		t.Fatalf("encode: %v", err)
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(plain.Bytes()); err != nil {
		t.Fatalf("gzip: %v", err)
	}
	w.Close()
	return gz.Bytes()
}

// blockEntityFreqCount decodes structure NBT and returns the number of blocks
// that carry a block-entity "nbt" compound containing a "Frequency" list.
func blockEntityFreqCount(t *testing.T, data []byte) int {
	t.Helper()
	dec, err := decompressLimited(data)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	var root map[string]interface{}
	if _, err := nbt.NewDecoder(bytes.NewReader(dec)).Decode(&root); err != nil {
		t.Fatalf("decode: %v", err)
	}
	blocks, _ := root["blocks"].([]interface{})
	n := 0
	for _, b := range blocks {
		bm, _ := b.(map[string]interface{})
		be, ok := bm["nbt"].(map[string]interface{})
		if !ok {
			continue
		}
		if _, has := be["Frequency"]; has {
			n++
		}
	}
	return n
}

func TestReplacePalette_PreservesBlockEntities(t *testing.T) {
	src := buildStructureWithBE(t)
	if got := blockEntityFreqCount(t, src); got != 1 {
		t.Fatalf("sanity: source has %d block-entity frequencies, want 1", got)
	}

	// Replace an UNRELATED block (stone -> air); the redstone link is untouched.
	out, err := ReplacePalette(src, []ReplaceBlock{{OriginalID: "minecraft:stone", ReplacementID: "minecraft:air"}})
	if err != nil {
		t.Fatalf("ReplacePalette: %v", err)
	}
	if got := blockEntityFreqCount(t, out); got != 1 {
		t.Errorf("after ReplacePalette: %d block-entity frequencies survived, want 1 (block-entity NBT was stripped)", got)
	}
}
