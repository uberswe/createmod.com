package abtest

import (
	"context"
	"math/rand"
	"net/http"
	"time"
)

const (
	trendingCookieName = "cm_trending_variant"
	trendingCookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
)

type trendingContextKey struct{}

// AssignTrendingVariant reads the trending variant from the request cookie or query param override.
// If no valid assignment exists, a new one is randomly chosen based on weights.
// The cookie is set on the response for sticky assignment (not for query-param overrides).
func AssignTrendingVariant(r *http.Request, w http.ResponseWriter, variants []TrendingVariant) *TrendingVariant {
	if len(variants) == 0 {
		return nil
	}

	// Query param override — no cookie set.
	if override := r.URL.Query().Get("trending_variant"); override != "" {
		if v := TrendingVariantByName(variants, override); v != nil {
			return v
		}
	}

	// Check existing cookie.
	if cookie, err := r.Cookie(trendingCookieName); err == nil {
		if v := TrendingVariantByName(variants, cookie.Value); v != nil {
			return v
		}
		// Invalid cookie value — fall through to reassign.
	}

	// Weighted random assignment.
	v := trendingWeightedRandom(variants)
	http.SetCookie(w, &http.Cookie{
		Name:     trendingCookieName,
		Value:    v.Name,
		Path:     "/",
		MaxAge:   trendingCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false, // JS needs to read it for GA
	})
	return v
}

// trendingWeightedRandom selects a trending variant based on relative weights.
func trendingWeightedRandom(variants []TrendingVariant) *TrendingVariant {
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

// ContextWithTrendingVariant stores the trending variant in the request context.
func ContextWithTrendingVariant(ctx context.Context, v *TrendingVariant) context.Context {
	return context.WithValue(ctx, trendingContextKey{}, v)
}

// TrendingVariantFromContext retrieves the trending variant from the request context.
func TrendingVariantFromContext(ctx context.Context) *TrendingVariant {
	v, _ := ctx.Value(trendingContextKey{}).(*TrendingVariant)
	return v
}
