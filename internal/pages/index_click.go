package pages

import (
	"createmod/internal/metrics"
	"createmod/internal/server"
	"encoding/json"
	"log/slog"
	"net/http"
)

type indexClickRequest struct {
	SchematicID string `json:"schematic_id"`
	Position    int    `json:"position"`
	Section     string `json:"section"`
}

// IndexClickHandler records a trending schematic click for analytics.
// POST /api/index/click
func IndexClickHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var req indexClickRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return &server.APIError{Status: 400, Message: "invalid request body"}
		}

		metrics.IndexClicks.WithLabelValues("7", req.Section).Inc()

		slog.Info("index",
			"event", "index_click",
			"window_days", "7",
			"schematic_id", req.SchematicID,
			"position", req.Position,
			"section", req.Section,
		)

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}
