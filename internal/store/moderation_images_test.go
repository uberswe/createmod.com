package store

import (
	"reflect"
	"testing"
)

func TestImageVisibility(t *testing.T) {
	s := &Schematic{
		Gallery:        []string{"a", "b", "c", "d"},
		HeldImages:     []string{"b", "x"}, // x is held but not in gallery (e.g. a held featured)
		RemovedImages:  []string{"c"},
		RotationImages: []string{"a", "b"},
	}

	// Visitor: exclude held (b) and removed (c).
	if got, want := VisibleGallery(s, false), []string{"a", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("VisibleGallery(visitor) = %v, want %v", got, want)
	}
	// Owner/admin: keep held (b), still exclude removed (c).
	if got, want := VisibleGallery(s, true), []string{"a", "b", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("VisibleGallery(owner) = %v, want %v", got, want)
	}
	// Rotation filtered the same way for visitors (b held).
	if got, want := VisibleRotationImages(s, false), []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("VisibleRotationImages(visitor) = %v, want %v", got, want)
	}
	// Held tiles for the owner: held minus removed (x has no removal, b kept).
	if got, want := HeldGallery(s), []string{"b", "x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("HeldGallery = %v, want %v", got, want)
	}
	if !IsImageHeld(s, "b") {
		t.Error("IsImageHeld(b) = false, want true")
	}
	if IsImageHeld(s, "a") {
		t.Error("IsImageHeld(a) = true, want false")
	}
	// A removed image is not 'held' even if it also appears in held list.
	s2 := &Schematic{HeldImages: []string{"z"}, RemovedImages: []string{"z"}}
	if IsImageHeld(s2, "z") {
		t.Error("IsImageHeld(z) with z removed = true, want false")
	}
}
