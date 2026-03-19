package search

import (
	"context"
	"createmod/internal/models"
	"testing"
)

// Compile-time interface check.
var _ SearchEngine = (*BleveEngine)(nil)

func TestBleveEngine_VariantA_NoAIDescription(t *testing.T) {
	svc := NewEmpty(nil)
	svc.BuildIndex([]models.Schematic{
		{
			ID:            "s1",
			Title:         "Simple Farm",
			Content:       "A basic farm",
			AIDescription: "automated wheat harvesting machine with redstone",
			Rating:        "4.0",
		},
	}, nil)

	// Variant A (base): should NOT find the schematic when searching for AI-only text.
	baseEngine := NewBleveEngine(svc, true)
	results, err := baseEngine.Search(context.Background(), SearchQuery{
		Term:     "automated wheat harvesting",
		Order:    BestMatchOrder,
		Rating:   -1,
		Category: "all",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("variant A should not match on AIDescription text, got %d results", len(results))
	}

	// Variant B (AI): should find the schematic.
	aiEngine := NewBleveEngine(svc, false)
	results, err = aiEngine.Search(context.Background(), SearchQuery{
		Term:     "automated wheat harvesting",
		Order:    BestMatchOrder,
		Rating:   -1,
		Category: "all",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("variant B should match on AIDescription text, got %d results", len(results))
	}
}

func TestBleveEngine_VariantA_MatchesTitle(t *testing.T) {
	svc := NewEmpty(nil)
	svc.BuildIndex([]models.Schematic{
		{
			ID:      "s1",
			Title:   "Compact Cobblestone Generator",
			Content: "Generates cobblestone automatically",
			Rating:  "3.5",
		},
	}, nil)

	baseEngine := NewBleveEngine(svc, true)
	results, err := baseEngine.Search(context.Background(), SearchQuery{
		Term:     "cobblestone generator",
		Order:    BestMatchOrder,
		Rating:   -1,
		Category: "all",
	})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("variant A should match on title text, got %d results", len(results))
	}
}

func TestBleveEngine_Ready(t *testing.T) {
	svc := NewEmpty(nil)
	engine := NewBleveEngine(svc, false)
	if engine.Ready() {
		t.Error("engine should not be ready before index build")
	}

	svc.BuildIndex([]models.Schematic{
		{ID: "s1", Title: "Test", Content: "test", Rating: "3.0"},
	}, nil)
	if !engine.Ready() {
		t.Error("engine should be ready after index build")
	}
}
