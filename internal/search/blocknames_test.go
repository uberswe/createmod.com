package search

import (
	"testing"
)

func TestExtractBlockNames_Valid(t *testing.T) {
	json := `[{"block_id":"create:brass_casing","count":12},{"block_id":"minecraft:oak_planks","count":4}]`
	names := ExtractBlockNames(json)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "Brass Casing" {
		t.Errorf("expected 'Brass Casing', got %q", names[0])
	}
	if names[1] != "Oak Planks" {
		t.Errorf("expected 'Oak Planks', got %q", names[1])
	}
}

func TestExtractBlockNames_Empty(t *testing.T) {
	if names := ExtractBlockNames(""); names != nil {
		t.Errorf("expected nil for empty string, got %v", names)
	}
}

func TestExtractBlockNames_EmptyArray(t *testing.T) {
	if names := ExtractBlockNames("[]"); names != nil {
		t.Errorf("expected nil for empty array, got %v", names)
	}
}

func TestExtractBlockNames_Malformed(t *testing.T) {
	if names := ExtractBlockNames("not json"); names != nil {
		t.Errorf("expected nil for malformed JSON, got %v", names)
	}
}

func TestExtractBlockNames_Deduplication(t *testing.T) {
	json := `[{"block_id":"create:brass_casing","count":5},{"block_id":"create:brass_casing","count":3}]`
	names := ExtractBlockNames(json)
	if len(names) != 1 {
		t.Fatalf("expected 1 deduplicated name, got %d", len(names))
	}
	if names[0] != "Brass Casing" {
		t.Errorf("expected 'Brass Casing', got %q", names[0])
	}
}

func TestExtractBlockNames_NoNamespace(t *testing.T) {
	json := `[{"block_id":"stone","count":1}]`
	names := ExtractBlockNames(json)
	if len(names) != 1 {
		t.Fatalf("expected 1 name, got %d", len(names))
	}
	if names[0] != "Stone" {
		t.Errorf("expected 'Stone', got %q", names[0])
	}
}
