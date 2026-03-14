package cache

import (
	"createmod/internal/models"
	"testing"
	"time"
)

func TestNew_InMemoryOnly(t *testing.T) {
	s := New()
	if s.c == nil {
		t.Fatal("expected in-memory cache to be initialized")
	}
	if s.redis != nil {
		t.Fatal("expected redis to be nil for in-memory-only service")
	}
}

func TestSetGetInt(t *testing.T) {
	s := New()
	s.SetInt("count", 42)
	v, ok := s.GetInt("count")
	if !ok {
		t.Fatal("expected to find 'count' in cache")
	}
	if v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
}

func TestGetInt_Missing(t *testing.T) {
	s := New()
	_, ok := s.GetInt("missing")
	if ok {
		t.Fatal("expected 'missing' to not be found")
	}
}

func TestSetGetFloat(t *testing.T) {
	s := New()
	s.SetFloat("score", 3.14)
	v, ok := s.GetFloat("score")
	if !ok {
		t.Fatal("expected to find 'score' in cache")
	}
	if v != 3.14 {
		t.Errorf("expected 3.14, got %f", v)
	}
}

func TestSetGetString(t *testing.T) {
	s := New()
	s.SetString("key", "value")
	v, ok := s.GetString("key")
	if !ok {
		t.Fatal("expected to find 'key' in cache")
	}
	if v != "value" {
		t.Errorf("expected 'value', got %q", v)
	}
}

func TestSetGetSchematic(t *testing.T) {
	s := New()
	schem := models.Schematic{
		ID:    "test-123",
		Title: "Test Schematic",
		Views: 100,
	}
	s.SetSchematic("s:test-123", schem)
	v, ok := s.GetSchematic("s:test-123")
	if !ok {
		t.Fatal("expected to find schematic in cache")
	}
	if v.ID != "test-123" {
		t.Errorf("expected ID 'test-123', got %q", v.ID)
	}
	if v.Title != "Test Schematic" {
		t.Errorf("expected Title 'Test Schematic', got %q", v.Title)
	}
	if v.Views != 100 {
		t.Errorf("expected Views 100, got %d", v.Views)
	}
}

func TestSetGetSchematics(t *testing.T) {
	s := New()
	schematics := []models.Schematic{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
	}
	s.SetSchematics("list", schematics)
	v, ok := s.GetSchematics("list")
	if !ok {
		t.Fatal("expected to find schematics in cache")
	}
	if len(v) != 2 {
		t.Errorf("expected 2 schematics, got %d", len(v))
	}
}

func TestSetGetCategories(t *testing.T) {
	s := New()
	cats := []models.SchematicCategory{
		{ID: "1", Key: "farm", Name: "Farm"},
		{ID: "2", Key: "train", Name: "Train"},
	}
	s.SetCategories("cats", cats, 10*time.Minute)
	v, ok := s.GetCategories("cats")
	if !ok {
		t.Fatal("expected to find categories in cache")
	}
	if len(v) != 2 {
		t.Errorf("expected 2 categories, got %d", len(v))
	}
}

func TestSetGetTagWithCount(t *testing.T) {
	s := New()
	tags := []models.SchematicTagWithCount{
		{ID: "1", Key: "farm", Name: "Farm", Count: 10},
	}
	s.SetTagWithCount("tags", tags)
	v, ok := s.GetTagWithCount("tags")
	if !ok {
		t.Fatal("expected to find tags in cache")
	}
	if len(v) != 1 {
		t.Errorf("expected 1 tag, got %d", len(v))
	}
	if v[0].Count != 10 {
		t.Errorf("expected count 10, got %d", v[0].Count)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.SetString("key", "value")
	s.Delete("key")
	_, ok := s.GetString("key")
	if ok {
		t.Fatal("expected 'key' to be deleted")
	}
}

func TestDeleteSchematic(t *testing.T) {
	s := New()
	s.SetSchematic("s:1", models.Schematic{ID: "1"})
	s.DeleteSchematic("s:1")
	_, ok := s.GetSchematic("s:1")
	if ok {
		t.Fatal("expected schematic to be deleted")
	}
}

func TestFlush(t *testing.T) {
	s := New()
	s.SetInt("a", 1)
	s.SetString("b", "x")
	s.Flush()
	_, ok1 := s.GetInt("a")
	_, ok2 := s.GetString("b")
	if ok1 || ok2 {
		t.Fatal("expected all keys to be flushed")
	}
}

func TestGenericSetGet(t *testing.T) {
	s := New()
	s.Set("custom", map[string]int{"a": 1})
	v, ok := s.Get("custom")
	if !ok {
		t.Fatal("expected to find 'custom' in cache")
	}
	m, ok := v.(map[string]int)
	if !ok {
		t.Fatal("expected map[string]int type")
	}
	if m["a"] != 1 {
		t.Errorf("expected a=1, got %d", m["a"])
	}
}

func TestSetWithTTL(t *testing.T) {
	s := New()
	s.SetWithTTL("short", "val", 1*time.Millisecond)
	// Value should be there immediately
	v, ok := s.Get("short")
	if !ok {
		t.Fatal("expected to find 'short' in cache immediately after set")
	}
	if v != "val" {
		t.Errorf("expected 'val', got %v", v)
	}
}

func TestClose_NoRedis(t *testing.T) {
	s := New()
	// Close should not panic when there's no Redis
	s.Close()
}

func TestKeyFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
		arg  string
		want string
	}{
		{"SchematicKey", SchematicKey, "abc", "schematic:abc"},
		{"ViewKey", ViewKey, "abc", "views:abc"},
		{"DownloadKey", DownloadKey, "abc", "downloads:abc"},
		{"RatingKey", RatingKey, "abc", "rating:abc"},
		{"RatingCountKey", RatingCountKey, "abc", "ratingCount:abc"},
		{"MinecraftVersionKey", MinecraftVersionKey, "1.20", "mcversion:1.20"},
		{"CreatemodVersionKey", CreatemodVersionKey, "0.5", "cmversion:0.5"},
		{"ModMetadataKey", ModMetadataKey, "create", "modmeta:create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(tt.arg)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
