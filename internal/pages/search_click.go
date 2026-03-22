package pages

import (
	"createmod/internal/metrics"
	"createmod/internal/server"
	"encoding/json"
	"log/slog"
	"net/http"
)

type searchClickRequest struct {
	Query    string `json:"query"`
	ResultID string `json:"result_id"`
	Position int    `json:"position"`
}

// SearchClickHandler records a search result click for analytics.
// POST /api/search/click
func SearchClickHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var req searchClickRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return &server.APIError{Status: 400, Message: "invalid request body"}
		}

		metrics.SearchClicks.WithLabelValues("meilisearch", "mods").Inc()

		slog.Info("search",
			"event", "search_click",
			"engine", "meilisearch",
			"index", "mods",
			"query", req.Query,
			"result_id", req.ResultID,
			"position", req.Position,
		)

		e.Response.WriteHeader(http.StatusNoContent)
		return nil
	}
}
