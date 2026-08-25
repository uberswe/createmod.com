package pages

import (
	"context"
	"testing"

	"createmod/internal/store"
)

// --- minimal in-memory fakes for the two stores the checklist logic touches ---

type memChecklist struct {
	items map[string][]*store.ModerationChecklistItem
	seq   int
}

func newMemChecklist() *memChecklist {
	return &memChecklist{items: map[string][]*store.ModerationChecklistItem{}}
}

func (m *memChecklist) Create(_ context.Context, it *store.ModerationChecklistItem) error {
	m.seq++
	cp := *it
	cp.ID = string(rune('a' + m.seq))
	if cp.Status == "" {
		cp.Status = store.ChecklistStatusOpen
	}
	m.items[it.SchematicID] = append(m.items[it.SchematicID], &cp)
	return nil
}
func (m *memChecklist) ListBySchematic(_ context.Context, id string) ([]store.ModerationChecklistItem, error) {
	return m.deref(id, false), nil
}
func (m *memChecklist) ListOpenBySchematic(_ context.Context, id string) ([]store.ModerationChecklistItem, error) {
	return m.deref(id, true), nil
}
func (m *memChecklist) CountOpenBySchematic(_ context.Context, id string) (int, error) {
	return len(m.deref(id, true)), nil
}
func (m *memChecklist) Resolve(_ context.Context, itemID string) error {
	for _, list := range m.items {
		for _, it := range list {
			if it.ID == itemID {
				it.Status = store.ChecklistStatusResolved
			}
		}
	}
	return nil
}
func (m *memChecklist) ResolveOpenByKind(_ context.Context, id, kind string) (int, error) {
	n := 0
	for _, it := range m.items[id] {
		if it.Kind == kind && it.Status == store.ChecklistStatusOpen {
			it.Status = store.ChecklistStatusResolved
			n++
		}
	}
	return n, nil
}
func (m *memChecklist) DeleteBySchematic(_ context.Context, id string) error {
	delete(m.items, id)
	return nil
}
func (m *memChecklist) deref(id string, openOnly bool) []store.ModerationChecklistItem {
	var out []store.ModerationChecklistItem
	for _, it := range m.items[id] {
		if openOnly && it.Status != store.ChecklistStatusOpen {
			continue
		}
		out = append(out, *it)
	}
	return out
}

// memSchematics implements only the methods the checklist logic calls.
type memSchematics struct {
	store.SchematicStore
	row *store.Schematic
}

func (m *memSchematics) GetByID(_ context.Context, id string) (*store.Schematic, error) {
	if m.row != nil && m.row.ID == id {
		cp := *m.row
		return &cp, nil
	}
	return nil, nil
}
func (m *memSchematics) Update(_ context.Context, s *store.Schematic) error {
	m.row = s
	return nil
}

func TestChecklistLifecycle(t *testing.T) {
	ctx := context.Background()
	cl := newMemChecklist()
	sch := &memSchematics{row: &store.Schematic{ID: "s1", ModerationState: store.ModerationAutoReview}}
	appStore := &store.Store{Schematics: sch, ModerationChecklist: cl}

	// 1. Quality failure enters published_limited and creates one description item.
	if !EnterPublishedLimited(ctx, appStore, sch.row, "Too short. Add detail.") {
		t.Fatal("EnterPublishedLimited should transition auto_review -> published_limited")
	}
	if sch.row.ModerationState != store.ModerationPublishedLimited {
		t.Fatalf("state = %q, want published_limited", sch.row.ModerationState)
	}
	if n, _ := cl.CountOpenBySchematic(ctx, "s1"); n != 1 {
		t.Fatalf("open items = %d, want 1", n)
	}

	// 2. Re-running the same check must not duplicate the item (dedup by kind).
	CreateAutoChecklistItem(ctx, appStore, "s1", store.ChecklistKindDescription, "still short")
	if n, _ := cl.CountOpenBySchematic(ctx, "s1"); n != 1 {
		t.Fatalf("after re-check open items = %d, want 1 (deduped)", n)
	}

	// 3. Promotion is blocked while an item is open.
	if promoted, _ := PromoteIfChecklistResolved(ctx, appStore, "s1"); promoted {
		t.Fatal("must not promote while a checklist item is open")
	}

	// 4. Resolving the description item and rechecking promotes to published.
	if n, _ := cl.ResolveOpenByKind(ctx, "s1", store.ChecklistKindDescription); n != 1 {
		t.Fatalf("resolved %d items, want 1", n)
	}
	promoted, err := PromoteIfChecklistResolved(ctx, appStore, "s1")
	if err != nil {
		t.Fatalf("promote error: %v", err)
	}
	if !promoted {
		t.Fatal("should promote once no open items remain")
	}
	if sch.row.ModerationState != store.ModerationPublished {
		t.Fatalf("final state = %q, want published", sch.row.ModerationState)
	}

	// 5. Idempotent: a second promote call is a no-op (already published).
	if promoted, _ := PromoteIfChecklistResolved(ctx, appStore, "s1"); promoted {
		t.Fatal("second promote should be a no-op")
	}
}
