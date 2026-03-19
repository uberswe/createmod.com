package abtest

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssignTrendingVariant_StickyAssignment(t *testing.T) {
	variants := DefaultTrendingVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	v := AssignTrendingVariant(r, w, variants)
	if v == nil {
		t.Fatal("expected non-nil variant")
	}

	// Extract cookie from response.
	resp := w.Result()
	cookies := resp.Cookies()
	var found *http.Cookie
	for _, c := range cookies {
		if c.Name == trendingCookieName {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("expected cm_trending_variant cookie to be set")
	}
	if found.Value != v.Name {
		t.Errorf("cookie value %q != variant name %q", found.Value, v.Name)
	}

	// Second request with cookie should return same variant.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(found)

	v2 := AssignTrendingVariant(r2, w2, variants)
	if v2.Name != v.Name {
		t.Errorf("sticky assignment failed: got %q, want %q", v2.Name, v.Name)
	}

	// No new cookie should be set on the second request.
	resp2 := w2.Result()
	if len(resp2.Cookies()) > 0 {
		t.Error("expected no new cookie on sticky assignment")
	}
}

func TestAssignTrendingVariant_QueryParamOverride(t *testing.T) {
	variants := DefaultTrendingVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?trending_variant=T60", nil)

	v := AssignTrendingVariant(r, w, variants)
	if v == nil {
		t.Fatal("expected non-nil variant")
	}
	if v.Name != "T60" {
		t.Errorf("expected variant T60 from override, got %q", v.Name)
	}

	// Query param override should NOT set a cookie.
	resp := w.Result()
	for _, c := range resp.Cookies() {
		if c.Name == trendingCookieName {
			t.Error("query param override should not set a cookie")
		}
	}
}

func TestAssignTrendingVariant_InvalidCookieReassigns(t *testing.T) {
	variants := DefaultTrendingVariants()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: trendingCookieName, Value: "INVALID"})

	v := AssignTrendingVariant(r, w, variants)
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
		if c.Name == trendingCookieName {
			found = true
		}
	}
	if !found {
		t.Error("expected new cookie after invalid cookie reassignment")
	}
}

func TestTrendingWeightedRandom_Distribution(t *testing.T) {
	variants := []TrendingVariant{
		{Name: "T7", WindowDays: 7, Weight: 50},
		{Name: "T15", WindowDays: 15, Weight: 50},
	}

	counts := map[string]int{}
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		v := trendingWeightedRandom(variants)
		counts[v.Name]++
	}

	// Each should be roughly 50% — allow 10% tolerance.
	for _, name := range []string{"T7", "T15"} {
		ratio := float64(counts[name]) / float64(iterations)
		if math.Abs(ratio-0.5) > 0.1 {
			t.Errorf("variant %s: expected ~50%%, got %.1f%%", name, ratio*100)
		}
	}
}

func TestTrendingContextRoundTrip(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	v := &TrendingVariant{Name: "T60", WindowDays: 60}
	ctx := ContextWithTrendingVariant(r.Context(), v)

	got := TrendingVariantFromContext(ctx)
	if got == nil || got.Name != "T60" {
		t.Errorf("context round-trip failed: got %v", got)
	}
}
