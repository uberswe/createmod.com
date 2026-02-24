package pages

import (
	"createmod/internal/search"
	"encoding/json"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// SearchSuggestHandler returns JSON autocomplete suggestions for the search input.
// GET /api/search/suggest?q=...
func SearchSuggestHandler(searchService *search.Service) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		if len(q) < 2 {
			e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = e.Response.Write([]byte("[]"))
			return nil
		}

		suggestions := searchService.Suggest(q, 8)
		if suggestions == nil {
			suggestions = []search.Suggestion{}
		}

		e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
		return json.NewEncoder(e.Response).Encode(suggestions)
	}
}
