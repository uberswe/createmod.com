package generator

import (
	"encoding/json"
	"testing"
)

// The build guide POSTs only the non-default params it decoded from a share
// hash. Seeding the decode with defaults must keep hollow=true (omitted because
// it is the default) instead of letting JSON's zero value make it false, which
// generated a solid balloon (issue #1603).
func TestDefaultBalloonParams_PartialJSONKeepsHollow(t *testing.T) {
	p := DefaultBalloonParams()
	if !p.Hollow {
		t.Fatal("default hollow should be true")
	}
	// Guide-style body: only a non-default field, hollow omitted.
	if err := json.Unmarshal([]byte(`{"lengthX":40}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Hollow {
		t.Error("hollow flipped to false after partial overlay — guide would render solid")
	}
	if p.LengthX != 40 || p.WidthZ != 18 || p.HeightY != 16 {
		t.Errorf("dims = %d/%d/%d, want 40/18/16 (lengthX overridden, rest default)", p.LengthX, p.WidthZ, p.HeightY)
	}

	// An explicit hollow:false must still win.
	p2 := DefaultBalloonParams()
	if err := json.Unmarshal([]byte(`{"hollow":false}`), &p2); err != nil {
		t.Fatal(err)
	}
	if p2.Hollow {
		t.Error("explicit hollow:false should override the default")
	}

	// And the generated builds must differ: hollow shell vs solid fill.
	hollowRes, err := GenerateBalloon(DefaultBalloonParams())
	if err != nil {
		t.Fatal(err)
	}
	solid := DefaultBalloonParams()
	solid.Hollow = false
	solidRes, err := GenerateBalloon(solid)
	if err != nil {
		t.Fatal(err)
	}
	count := func(r *GenerateResult) int {
		n := 0
		for _, b := range r.Blocks {
			if b.Type != BlockAir {
				n++
			}
		}
		return n
	}
	if count(hollowRes) >= count(solidRes) {
		t.Errorf("hollow balloon (%d blocks) should have fewer blocks than solid (%d)", count(hollowRes), count(solidRes))
	}
}
