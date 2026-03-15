package nbtparser

import (
	"bytes"
	"compress/gzip"
	"testing"

	nbt "github.com/Tnze/go-mc/nbt"
)

// buildTestNBT creates a minimal gzip-compressed schematic NBT with the given palette entries.
func buildTestNBT(t *testing.T, paletteEntries []map[string]interface{}) []byte {
	t.Helper()

	root := map[string]interface{}{
		"palette": paletteEntries,
		"blocks":  []interface{}{},
		"size":    []int32{1, 1, 1},
		"Version": int32(2),
	}

	var nbtBuf bytes.Buffer
	encoder := nbt.NewEncoder(&nbtBuf)
	if err := encoder.Encode(root, "Schematic"); err != nil {
		t.Fatalf("failed to encode test NBT: %v", err)
	}

	var gzBuf bytes.Buffer
	gzw := gzip.NewWriter(&gzBuf)
	if _, err := gzw.Write(nbtBuf.Bytes()); err != nil {
		t.Fatalf("failed to gzip: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("failed to close gzip: %v", err)
	}
	return gzBuf.Bytes()
}

// decodePalette decompresses and decodes NBT to extract the palette entries.
func decodePalette(t *testing.T, data []byte) []map[string]interface{} {
	t.Helper()

	decompressed, err := decompressLimited(data)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	var decoded interface{}
	decoder := nbt.NewDecoder(bytes.NewReader(decompressed))
	_, err = decoder.Decode(&decoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	rootMap := decoded.(map[string]interface{})
	paletteRaw := rootMap["palette"].([]interface{})

	result := make([]map[string]interface{}, len(paletteRaw))
	for i, entry := range paletteRaw {
		result[i] = entry.(map[string]interface{})
	}
	return result
}

func TestReplacePalette_BasicReplacement(t *testing.T) {
	palette := []map[string]interface{}{
		{"Name": "minecraft:oak_planks"},
		{"Name": "minecraft:stone"},
	}
	data := buildTestNBT(t, palette)

	result, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:oak_planks", ReplacementID: "minecraft:spruce_planks"},
	})
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	entries := decodePalette(t, result)
	if entries[0]["Name"] != "minecraft:spruce_planks" {
		t.Errorf("expected minecraft:spruce_planks, got %v", entries[0]["Name"])
	}
	if entries[1]["Name"] != "minecraft:stone" {
		t.Errorf("stone should be unchanged, got %v", entries[1]["Name"])
	}
}

func TestReplacePalette_Removal(t *testing.T) {
	palette := []map[string]interface{}{
		{
			"Name":       "minecraft:oak_stairs",
			"Properties": map[string]interface{}{"facing": "north"},
		},
	}
	data := buildTestNBT(t, palette)

	result, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:oak_stairs", ReplacementID: ""}, // remove
	})
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	entries := decodePalette(t, result)
	if entries[0]["Name"] != "minecraft:air" {
		t.Errorf("expected minecraft:air, got %v", entries[0]["Name"])
	}
	if _, has := entries[0]["Properties"]; has {
		t.Errorf("properties should be removed for air replacement")
	}
}

func TestReplacePalette_SameFamilyKeepsProperties(t *testing.T) {
	palette := []map[string]interface{}{
		{
			"Name":       "minecraft:oak_stairs",
			"Properties": map[string]interface{}{"facing": "north", "half": "bottom"},
		},
	}
	data := buildTestNBT(t, palette)

	result, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:oak_stairs", ReplacementID: "minecraft:spruce_stairs"},
	})
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	entries := decodePalette(t, result)
	if entries[0]["Name"] != "minecraft:spruce_stairs" {
		t.Errorf("expected minecraft:spruce_stairs, got %v", entries[0]["Name"])
	}
	props, has := entries[0]["Properties"]
	if !has {
		t.Fatalf("properties should be kept for same-family replacement")
	}
	propsMap := props.(map[string]interface{})
	if propsMap["facing"] != "north" {
		t.Errorf("expected facing=north, got %v", propsMap["facing"])
	}
}

func TestReplacePalette_DifferentFamilyStripsProperties(t *testing.T) {
	palette := []map[string]interface{}{
		{
			"Name":       "minecraft:oak_stairs",
			"Properties": map[string]interface{}{"facing": "north"},
		},
	}
	data := buildTestNBT(t, palette)

	result, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:oak_stairs", ReplacementID: "minecraft:stone"},
	})
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	entries := decodePalette(t, result)
	if entries[0]["Name"] != "minecraft:stone" {
		t.Errorf("expected minecraft:stone, got %v", entries[0]["Name"])
	}
	if _, has := entries[0]["Properties"]; has {
		t.Errorf("properties should be stripped for different-family replacement")
	}
}

func TestReplacePalette_EmptyReplacements(t *testing.T) {
	palette := []map[string]interface{}{
		{"Name": "minecraft:stone"},
	}
	data := buildTestNBT(t, palette)

	result, err := ReplacePalette(data, nil)
	if err != nil {
		t.Fatalf("ReplacePalette failed: %v", err)
	}

	// Should return original data unchanged
	if !bytes.Equal(result, data) {
		t.Error("empty replacements should return original data unchanged")
	}
}

func TestReplacePalette_InvalidBlockID(t *testing.T) {
	palette := []map[string]interface{}{
		{"Name": "minecraft:stone"},
	}
	data := buildTestNBT(t, palette)

	_, err := ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "not_valid", ReplacementID: "minecraft:stone"},
	})
	if err == nil {
		t.Error("expected error for invalid block ID")
	}

	_, err = ReplacePalette(data, []ReplaceBlock{
		{OriginalID: "minecraft:stone", ReplacementID: "not_valid"},
	})
	if err == nil {
		t.Error("expected error for invalid replacement block ID")
	}
}

func TestBlockFamilySuffix(t *testing.T) {
	tests := []struct {
		blockID  string
		expected string
	}{
		{"minecraft:oak_stairs", "_stairs"},
		{"minecraft:spruce_slab", "_slab"},
		{"minecraft:oak_planks", "_planks"},
		{"minecraft:dark_oak_log", "_log"},
		{"minecraft:stone", ""},
		{"minecraft:dirt", ""},
		{"create:brass_fence", "_fence"},
	}

	for _, tc := range tests {
		got := blockFamilySuffix(tc.blockID)
		if got != tc.expected {
			t.Errorf("blockFamilySuffix(%q) = %q, want %q", tc.blockID, got, tc.expected)
		}
	}
}

func TestSameBlockFamily(t *testing.T) {
	if !sameBlockFamily("minecraft:oak_stairs", "minecraft:spruce_stairs") {
		t.Error("oak_stairs and spruce_stairs should be same family")
	}
	if sameBlockFamily("minecraft:oak_stairs", "minecraft:stone") {
		t.Error("oak_stairs and stone should not be same family")
	}
	if sameBlockFamily("minecraft:stone", "minecraft:dirt") {
		t.Error("stone and dirt should not be same family")
	}
}
