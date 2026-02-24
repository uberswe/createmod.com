package search

import (
	"testing"
	"time"
)

// newTestService creates a Service with a minimal in-memory index for testing Suggest.
func newTestService(schematics []schematicIndex) *Service {
	s := &Service{
		index: schematics,
	}
	return s
}

func Test_Suggest_Returns_Matching_Titles(t *testing.T) {
	s := newTestService([]schematicIndex{
		{ID: "1", Title: "Iron Farm", Tags: []string{"Farm"}, Categories: []string{"Automation"}, Created: time.Now()},
		{ID: "2", Title: "Gold Farm", Tags: []string{"Farm"}, Categories: []string{"Automation"}, Created: time.Now()},
		{ID: "3", Title: "Train Station", Tags: []string{"Transport"}, Categories: []string{"Rail"}, Created: time.Now()},
	})

	results := s.Suggest("iron", 8)
	if len(results) == 0 {
		t.Fatal("expected at least one suggestion for 'iron'")
	}
	found := false
	for _, r := range results {
		if r.Text == "Iron Farm" && r.Type == "schematic" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Iron Farm' schematic suggestion, got %+v", results)
	}
}

func Test_Suggest_Returns_Matching_Tags(t *testing.T) {
	s := newTestService([]schematicIndex{
		{ID: "1", Title: "Iron Farm", Tags: []string{"Farm", "Iron"}, Categories: []string{"Automation"}, Created: time.Now()},
	})

	results := s.Suggest("farm", 8)
	foundTag := false
	for _, r := range results {
		if r.Type == "tag" && r.Text == "Farm" {
			foundTag = true
		}
	}
	if !foundTag {
		t.Errorf("expected tag suggestion for 'farm', got %+v", results)
	}
}

func Test_Suggest_Limits_Results(t *testing.T) {
	items := make([]schematicIndex, 20)
	for i := range items {
		items[i] = schematicIndex{
			ID:    string(rune('a' + i)),
			Title: "Farm Design " + string(rune('A'+i)),
			Tags:  []string{"Farm"},
		}
	}
	s := newTestService(items)

	results := s.Suggest("farm", 5)
	if len(results) > 5 {
		t.Errorf("expected at most 5 results, got %d", len(results))
	}
}

func Test_Suggest_Empty_Query(t *testing.T) {
	s := newTestService([]schematicIndex{
		{ID: "1", Title: "Iron Farm", Tags: []string{"Farm"}, Created: time.Now()},
	})

	results := s.Suggest("", 8)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}

	results = s.Suggest("a", 8)
	if len(results) != 0 {
		t.Errorf("expected no results for single-char query, got %d", len(results))
	}
}
