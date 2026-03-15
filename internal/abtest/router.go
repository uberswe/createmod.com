package abtest

import (
	"context"
	"createmod/internal/search"
	"log/slog"
)

// VariantRouter maps variant names to search engines and provides
// fallback logic when a target engine is unhealthy.
type VariantRouter struct {
	engines  map[string]search.SearchEngine
	fallback search.SearchEngine // BleveBase (variant B) — always available
}

// NewVariantRouter creates a router with the given engine map and fallback.
func NewVariantRouter(engines map[string]search.SearchEngine, fallback search.SearchEngine) *VariantRouter {
	return &VariantRouter{
		engines:  engines,
		fallback: fallback,
	}
}

// GetEngine returns the search engine for the given variant.
// If the target engine is not ready, falls back to the Bleve base engine.
func (vr *VariantRouter) GetEngine(v *Variant) search.SearchEngine {
	if v == nil {
		return vr.fallback
	}
	eng, ok := vr.engines[v.Name]
	if !ok {
		slog.Warn("abtest: unknown variant, using fallback", "variant", v.Name)
		return vr.fallback
	}
	if !eng.Ready() {
		slog.Warn("abtest: engine not ready, using fallback", "variant", v.Name)
		return vr.fallback
	}
	return eng
}

// Fallback returns the fallback engine (Bleve with AI descriptions).
func (vr *VariantRouter) Fallback() search.SearchEngine {
	return vr.fallback
}

// HealthCheck runs health checks on all registered engines.
func (vr *VariantRouter) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error, len(vr.engines))
	for name, eng := range vr.engines {
		results[name] = eng.Health(ctx)
	}
	return results
}
