package pages

import (
	"encoding/json"
	"net/http"

	"createmod/internal/server"
)

// agentDiscoveryLinkHeader advertises API discovery resources to agents via
// an RFC 8288 Link header. Served on the homepage and the API docs page.
const agentDiscoveryLinkHeader = `</.well-known/api-catalog>; rel="api-catalog", </api>; rel="service-doc", </api/openapi.json>; rel="service-desc", </auth.md>; rel="help"`

// SetAgentDiscoveryLinkHeader adds the agent-discovery Link header to a response.
func SetAgentDiscoveryLinkHeader(e *server.RequestEvent) {
	e.Response.Header().Set("Link", agentDiscoveryLinkHeader)
}

// APICatalogHandler serves /.well-known/api-catalog (RFC 9727): a linkset
// document pointing agents at the machine-readable OpenAPI description and
// the human-readable API documentation.
func APICatalogHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		catalog := map[string]any{
			"linkset": []map[string]any{
				{
					"anchor": "https://createmod.com/api",
					"service-desc": []map[string]string{
						{"href": "https://createmod.com/api/openapi.json", "type": "application/vnd.oai.openapi+json"},
					},
					"service-doc": []map[string]string{
						{"href": "https://createmod.com/api", "type": "text/html"},
					},
				},
			},
		}
		body, err := json.Marshal(catalog)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to build catalog")
		}
		e.Response.Header().Set("Cache-Control", "public, max-age=86400")
		return e.Blob(http.StatusOK, "application/linkset+json", body)
	}
}
