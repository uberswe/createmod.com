package abtest

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssignVariant_StickyAssignment(t *testing.T) {
	variants := DefaultVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=test", nil)

	v := AssignVariant(r, w, variants)
	if v == nil {
		t.Fatal("expected non-nil variant")
	}

	// Extract cookie from response.
	resp := w.Result()
	cookies := resp.Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == cookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected cm_search_variant cookie to be set")
	}
	if found.Value != v.Name {
		t.Errorf("cookie value %q != variant name %q", found.Value, v.Name)
	}

	// Second request with cookie should return same variant.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/search?q=test", nil)
	r2.AddCookie(found)

	v2 := AssignVariant(r2, w2, variants)
	if v2.Name != v.Name {
		t.Errorf("sticky assignment failed: got %q, want %q", v2.Name, v.Name)
	}

	// No new cookie should be set on the second request.
	resp2 := w2.Result()
	if len(resp2.Cookies()) > 0 {
		t.Error("expected no new cookie on sticky assignment")
	}
}

func TestAssignVariant_QueryParamOverride(t *testing.T) {
	variants := DefaultVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=test&variant=E", nil)

	v := AssignVariant(r, w, variants)
	if v == nil {
		t.Fatal("expected non-nil variant")
	}
	if v.Name != "E" {
		t.Errorf("expected variant E from override, got %q", v.Name)
	}

	// Query param override should NOT set a cookie.
	resp := w.Result()
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			t.Error("query param override should not set a cookie")
		}
	}
}

func TestAssignVariant_InvalidCookieReassigns(t *testing.T) {
	variants := DefaultVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=test", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "INVALID"})

	v := AssignVariant(r, w, variants)
	if v == nil {
		t.Fatal("expected non-nil variant after reassignment")
	}
	if v.Name == "INVALID" {
		t.Error("invalid cookie value should trigger reassignment")
	}

	// A new cookie should be set.
	resp := w.Result()
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			found = true
		}
	}
	if !found {
		t.Error("expected new cookie after invalid cookie reassignment")
	}
}

func TestWeightedRandom_Distribution(t *testing.T) {
	variants := []Variant{
		{Name: "A", Weight: 50},
		{Name: "B", Weight: 50},
	}

	counts := map[string]int{}
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		v := weightedRandom(variants)
		counts[v.Name]++
	}

	// Each should be roughly 50% — allow 10% tolerance.
	for _, name := range []string{"A", "B"} {
		ratio := float64(counts[name]) / float64(iterations)
		if math.Abs(ratio-0.5) > 0.1 {
			t.Errorf("variant %s: expected ~50%%, got %.1f%%", name, ratio*100)
		}
	}
}

func TestVariantByName(t *testing.T) {
	variants := DefaultVariants()
	v := VariantByName(variants, "C")
	if v == nil {
		t.Fatal("expected non-nil variant C")
	}
	if v.Engine != "meilisearch" {
		t.Errorf("variant C engine = %q, want meilisearch", v.Engine)
	}

	v = VariantByName(variants, "Z")
	if v != nil {
		t.Error("expected nil for unknown variant Z")
	}
}

func TestContextRoundTrip(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	v := &Variant{Name: "D", Engine: "meilisearch", IndexLevel: "ai"}
	ctx := ContextWithVariant(r.Context(), v)

	got := VariantFromContext(ctx)
	if got == nil || got.Name != "D" {
		t.Errorf("context round-trip failed: got %v", got)
	}
}
