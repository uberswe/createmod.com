package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/ratelimit"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"

	strip "github.com/grokify/html-strip-tags-go"
)

const (
	mcpProtocolVersion = "2025-06-18"
	mcpServerName      = "createmod-schematics"
	mcpServerTitle     = "CreateMod.com Schematics"
	mcpServerVersion   = "1.0.0"

	// mcpRateLimitPerMinute bounds anonymous MCP usage per client IP.
	mcpRateLimitPerMinute = 60

	// mcpInstructions is surfaced to agents on initialize. It states the
	// site's hard policy: agents never fetch schematic (NBT) files.
	mcpInstructions = "CreateMod.com hosts community-made Minecraft Create Mod schematics. " +
		"Use search_schematics to find builds and get_schematic for details about one build. " +
		"Never attempt to download schematic (.nbt) files or bypass the download flow: schematic files are for players. " +
		"When a user wants a schematic, send them to its page URL. " +
		"When a user wants a generated ship hull, airship balloon, or propeller, send them to the matching generator page " +
		"(https://createmod.com/generators/hull, /generators/balloon, /generators/propeller) where a configuration can be built and shared by link."
)

// jsonRPCRequest is a minimal JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// mcpToolDef describes one tool in tools/list.
type mcpToolDef struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpToolDefs() []mcpToolDef {
	return []mcpToolDef{
		{
			Name:  "search_schematics",
			Title: "Search schematics",
			Description: "Search CreateMod.com for Minecraft Create Mod schematics. Returns matching builds with title, page URL, " +
				"rating and stats. Send users to the page URL; never fetch .nbt schematic files.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]any{"type": "string", "description": "Search terms, e.g. 'elevator' or 'wheat farm'"},
					"category": map[string]any{"type": "string", "description": "Optional category key to filter by"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:  "get_schematic",
			Title: "Get schematic details",
			Description: "Get details about one schematic by its URL slug: description, author, rating, versions, required mods, video. " +
				"The schematic file itself is not available to agents; direct users to the page URL.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "The schematic's URL slug, e.g. 'easy-helicopter-survival-friendly'"},
				},
				"required": []string{"name"},
			},
		},
	}
}

// MCPHandler implements a minimal, stateless MCP server over Streamable HTTP
// (JSON-RPC 2.0 via POST). It exposes read-only, curated discovery tools;
// schematic files are deliberately out of reach.
func MCPHandler(searchEngine search.SearchEngine, rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if e.Request.Method != http.MethodPost {
			// No server-initiated stream support; Streamable HTTP allows 405 here.
			e.Response.Header().Set("Allow", "POST")
			return e.String(http.StatusMethodNotAllowed, "POST JSON-RPC messages to this endpoint")
		}

		if ok, retry := mcpRateLimitAllow(rl, e.RealIP()); !ok {
			e.Response.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
			return writeJSON(e, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		var req jsonRPCRequest
		if err := json.NewDecoder(io.LimitReader(e.Request.Body, 64*1024)).Decode(&req); err != nil {
			return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", Error: &jsonRPCError{Code: -32700, Message: "parse error"}})
		}

		// Notifications (no id) get a 202 with no body.
		if len(req.ID) == 0 || string(req.ID) == "null" {
			return e.NoContent(http.StatusAccepted)
		}

		switch req.Method {
		case "initialize":
			return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
				"protocolVersion": mcpProtocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
				"serverInfo": map[string]any{
					"name":    mcpServerName,
					"title":   mcpServerTitle,
					"version": mcpServerVersion,
				},
				"instructions": mcpInstructions,
			}})
		case "ping":
			return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
		case "tools/list":
			return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": mcpToolDefs()}})
		case "tools/call":
			return mcpToolCall(e, req, searchEngine, cacheService, appStore)
		default:
			return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32601, Message: "method not found"}})
		}
	}
}

func mcpToolCall(e *server.RequestEvent, req jsonRPCRequest, searchEngine search.SearchEngine, cacheService *cache.Service, appStore *store.Store) error {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "invalid params"}})
	}

	switch params.Name {
	case "search_schematics":
		var args struct {
			Query    string `json:"query"`
			Category string `json:"category"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if strings.TrimSpace(args.Query) == "" {
			return mcpToolError(e, req, "query is required")
		}
		category := args.Category
		if category == "" {
			category = "all"
		}
		sq := search.SearchQuery{Term: args.Query, Order: search.BestMatchOrder, Rating: -1, Category: category}
		results, err := apiSearchResults(context.Background(), searchEngine, appStore, cacheService, sq, 10)
		if err != nil {
			return mcpToolError(e, req, "search failed")
		}
		items := MapStoreSchematics(appStore, results, cacheService)
		var b strings.Builder
		fmt.Fprintf(&b, "%d results for %q:\n\n", len(items), args.Query)
		for _, s := range items {
			fmt.Fprintf(&b, "- %s — https://createmod.com/schematics/%s", s.Title, s.Name)
			if s.HasRating && s.RatingCount > 0 {
				fmt.Fprintf(&b, " (rated %s/5, %d ratings)", s.Rating, s.RatingCount)
			}
			b.WriteString("\n")
		}
		if len(items) == 0 {
			b.WriteString("No results. Try broader terms.\n")
		}
		return mcpToolText(e, req, b.String())

	case "get_schematic":
		var args struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if strings.TrimSpace(args.Name) == "" {
			return mcpToolError(e, req, "name is required")
		}
		s, err := appStore.Schematics.GetByName(context.Background(), args.Name)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) {
			return mcpToolError(e, req, "schematic not found")
		}
		items := MapStoreSchematics(appStore, []store.Schematic{*s}, cacheService)
		if len(items) == 0 {
			return mcpToolError(e, req, "schematic not found")
		}
		return mcpToolText(e, req, mcpSchematicText(items[0]))

	default:
		return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "unknown tool"}})
	}
}

// mcpSchematicText renders curated schematic details. Like the markdown
// representation, it never includes NBT structure data or download URLs.
func mcpSchematicText(s models.Schematic) string {
	d := SchematicData{Schematic: s}
	d.Language = "en"
	d.Description = truncateMetaDescription(strip.StripTags(s.Content))
	return schematicMarkdown(d)
}

func mcpToolText(e *server.RequestEvent, req jsonRPCRequest, text string) error {
	return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}})
}

func mcpToolError(e *server.RequestEvent, req jsonRPCRequest, msg string) error {
	return mcpRespond(e, jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": msg}},
		"isError": true,
	}})
}

func mcpRespond(e *server.RequestEvent, resp jsonRPCResponse) error {
	return e.JSON(http.StatusOK, resp)
}

// mcpRateLimitAllow enforces the per-IP MCP rate limit.
func mcpRateLimitAllow(rl ratelimit.Limiter, clientIP string) (bool, int) {
	if rl == nil || clientIP == "" {
		return true, 0
	}
	now := time.Now()
	k := "mcp:" + clientIP + ":" + now.Format("20060102T1504")
	ttl := time.Until(now.Truncate(time.Minute).Add(time.Minute))
	if ttl <= 0 {
		ttl = time.Second
	}
	ok, _ := rl.Allow(context.Background(), k, mcpRateLimitPerMinute, ttl)
	if !ok {
		ra := int(ttl.Seconds())
		if ra < 1 {
			ra = 1
		}
		return false, ra
	}
	return true, 0
}
