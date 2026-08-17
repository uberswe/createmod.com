package jobs

import (
	"testing"

	"createmod/internal/schematic"
)

// mergeManifests must never hide a finding present in either file and must not
// double-count the same finding across the .nbt and its original (they describe
// the same build). This backs the guarantee that a preserved original can't be
// labeled safe while carrying something the normalized .nbt dropped.
func TestMergeManifests(t *testing.T) {
	cmd := schematic.Finding{Type: schematic.FindingCommandBlock, Pos: [3]int{1, 2, 3}, Detail: "/op @p"}
	spawner := schematic.Finding{Type: schematic.FindingSpawner, Pos: [3]int{4, 5, 6}}

	primary := &schematic.Manifest{
		InspectorVersion: schematic.InspectorVersion,
		Counts:           map[schematic.FindingType]int{schematic.FindingCommandBlock: 1},
		Findings:         []schematic.Finding{cmd},
		ModNamespaces:    []string{"create"},
	}
	// The original carries the same command block plus a spawner the .nbt lost.
	original := &schematic.Manifest{
		InspectorVersion: schematic.InspectorVersion,
		Counts:           map[schematic.FindingType]int{schematic.FindingCommandBlock: 1, schematic.FindingSpawner: 1},
		Findings:         []schematic.Finding{cmd, spawner},
		ModNamespaces:    []string{"create", "aeronautics"},
	}

	m := mergeManifests(primary, original)
	if got := m.Counts[schematic.FindingCommandBlock]; got != 1 {
		t.Errorf("command block count = %d, want 1 (per-type max, not summed)", got)
	}
	if got := m.Counts[schematic.FindingSpawner]; got != 1 {
		t.Errorf("spawner count = %d, want 1 (surfaced from the original)", got)
	}
	if len(m.Findings) != 2 {
		t.Fatalf("findings = %d, want 2 (deduped union)", len(m.Findings))
	}
	if !m.Notable() {
		t.Error("merged manifest should be notable")
	}
	if len(m.ModNamespaces) != 2 {
		t.Errorf("namespaces = %v, want create+aeronautics unioned", m.ModNamespaces)
	}

	// nil handling: merging with nil returns the other side unchanged.
	if mergeManifests(nil, original) != original {
		t.Error("merge(nil, x) should return x")
	}
	if mergeManifests(primary, nil) != primary {
		t.Error("merge(x, nil) should return x")
	}
}
