package abtest

import (
	"context"
	"math/rand"
	"net/http"
	"time"
)

const (
	cookieName = "cm_search_variant"
	cookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
)

type contextKey struct{}

// AssignVariant reads the variant from the request cookie or query param override.
// If no valid assignment exists, a new one is randomly chosen based on weights.
// The cookie is set on the response for sticky assignment (not for query-param overrides).
func AssignVariant(r *http.Request, w http.ResponseWriter, variants []Variant) *Variant {
	if len(variants) == 0 {
		return nil
	}

	// Query param override — no cookie set.
	if override := r.URL.Query().Get("variant"); override != "" {
		if v := VariantByName(variants, override); v != nil {
			return v
		}
	}

	// Check existing cookie.
	if cookie, err := r.Cookie(cookieName); err == nil {
		if v := VariantByName(variants, cookie.Value); v != nil {
			return v
		}
		// Invalid cookie value — fall through to reassign.
	}

	// Weighted random assignment.
	v := weightedRandom(variants)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    v.Name,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false, // JS needs to read it for GA
	})
	return v
}

// weightedRandom selects a variant based on relative weights.
func weightedRandom(variants []Variant) *Variant {
	total := 0
	for _, v := range variants {
		total += v.Weight
	}
	if total == 0 {
		return &variants[0]
	}
	//nolint:gosec // not security-sensitive
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := r.Intn(total)
	for i := range variants {
		n -= variants[i].Weight
		if n < 0 {
			return &variants[i]
		}
	}
	return &variants[len(variants)-1]
}

// ContextWithVariant stores the variant in the request context.
func ContextWithVariant(ctx context.Context, v *Variant) context.Context {
	return context.WithValue(ctx, contextKey{}, v)
}

// VariantFromContext retrieves the variant from the request context.
func VariantFromContext(ctx context.Context) *Variant {
	v, _ := ctx.Value(contextKey{}).(*Variant)
	return v
}
