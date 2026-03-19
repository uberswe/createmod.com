package pages

import (
	"createmod/internal/abtest"
	"createmod/internal/metrics"
	"createmod/internal/server"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type indexClickRequest struct {
	SchematicID string `json:"schematic_id"`
	Position    int    `json:"position"`
	Section     string `json:"section"`
}

// IndexClickHandler records a trending schematic click for A/B test analytics.
// POST /api/index/click
func IndexClickHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var req indexClickRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return &server.APIError{Status: 400, Message: "invalid request body"}
		}

		variant := abtest.TrendingVariantFromContext(e.Request.Context())
		variantName := ""
		windowDays := "30"
		if variant != nil {
			variantName = variant.Name
			windowDays = fmt.Sprintf("%d", variant.WindowDays)
		}

		metrics.IndexClicks.WithLabelValues(variantName, windowDays, req.Section).Inc()

		slog.Info("index",
			"event", "index_click",
			"variant", variantName,
			"window_days", windowDays,
			"schematic_id", req.SchematicID,
			"position", req.Position,
			"section", req.Section,
		)

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}
