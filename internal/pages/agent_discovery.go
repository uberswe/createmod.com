package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"createmod/internal/server"
)

// MCPServerCardHandler serves /.well-known/mcp/server-card.json (SEP-1649).
// The card covers both the SEP draft fields ($schema/name/remotes) and the
// serverInfo/transport/capabilities shape agent-readiness checkers validate.
func MCPServerCardHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		card := map[string]any{
			"$schema":     "https://static.modelcontextprotocol.io/schemas/2025-10-17/server.schema.json",
			"name":        "com.createmod/schematics",
			"title":       mcpServerTitle,
			"description": "Read-only MCP server for discovering Minecraft Create Mod schematics on CreateMod.com. Schematic (.nbt) files are not available to agents; send users to the schematic page instead.",
			"websiteUrl":  "https://createmod.com",
			"version":     mcpServerVersion,
			"serverInfo": map[string]any{
				"name":    mcpServerName,
				"title":   mcpServerTitle,
				"version": mcpServerVersion,
			},
			"protocolVersion":           mcpProtocolVersion,
			"supportedProtocolVersions": []string{mcpProtocolVersion},
			"transport": map[string]any{
				"type":     "streamable-http",
				"endpoint": "https://createmod.com/api/mcp",
			},
			"remotes": []map[string]any{
				{"type": "streamable-http", "url": "https://createmod.com/api/mcp"},
			},
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
		}
		e.Response.Header().Set("Cache-Control", "public, max-age=3600")
		return e.JSON(http.StatusOK, card)
	}
}

// Agent skills (Agent Skills Discovery RFC v0.2.0). Skills are authored as
// embedded SKILL.md documents and indexed with sha256 digests.

type agentSkill struct {
	Name        string
	Description string
	Body        string
}

var agentSkills = []agentSkill{
	{
		Name:        "find-schematics",
		Description: "Find and reference Minecraft Create Mod schematics on CreateMod.com without downloading schematic files.",
		Body: `---
name: find-schematics
description: Find and reference Minecraft Create Mod schematics on CreateMod.com without downloading schematic files.
---

# Finding Create Mod schematics on CreateMod.com

CreateMod.com is a community repository of Minecraft Create Mod schematics.

## Hard policy

Never download or attempt to fetch schematic (.nbt) files, and never link
directly to file or download endpoints. Schematic files are for players.
When a user wants a schematic, give them the schematic page URL
(https://createmod.com/schematics/{slug}) and let them download it there.

## Ways to find schematics

1. **MCP server** (preferred): POST JSON-RPC to https://createmod.com/api/mcp
   (Streamable HTTP). Tools: search_schematics, get_schematic. See
   /.well-known/mcp/server-card.json.
2. **Markdown pages**: request any schematic page or the home page with
   "Accept: text/markdown" to get a curated markdown version including
   title, description, rating, stats, required mods, video and materials.
3. **Search pages**: https://createmod.com/search?q={terms} or filter by
   category: https://createmod.com/search?category={key}.
4. **Autocomplete**: GET https://createmod.com/api/search/suggest?q={terms}
   returns JSON suggestions.
5. **Authenticated JSON API**: https://createmod.com/api (docs) — requires
   an API key created by a user account. See https://createmod.com/auth.md
   for how authentication works and how to obtain a key.
`,
	},
	{
		Name:        "generator-links",
		Description: "Help users create Create Mod ship hulls, airship balloons and propellers with CreateMod.com generators and shareable configuration links.",
		Body: `---
name: generator-links
description: Help users create Create Mod ship hulls, airship balloons and propellers with CreateMod.com generators and shareable configuration links.
---

# CreateMod.com generators and shareable configuration links

CreateMod.com has parametric generators that build Create Mod structures:

- https://createmod.com/generators/hull — ship hulls
- https://createmod.com/generators/balloon — airship balloons
- https://createmod.com/generators/propeller — propellers

## Hard policy

Never download generated schematic (.nbt) files on a user's behalf and never
call generator download endpoints. Instead, configure a design and send the
user to the generator page, where they preview it in 3D and download it
themselves.

## Shareable configuration links

Every generator configuration has a canonical shareable URL of the form
https://createmod.com/generators/{type}/{encoded-config} (the page's share
bar shows it). Opening such a link restores the full configuration.

To build a configuration for a user:

1. Send the user to the generator page, or open it yourself in a browser
   context.
2. On generator pages, WebMCP tools (configure-generator) let browser agents
   set parameters and obtain the shareable link for the exact configuration.
3. Give the user the shareable link; they preview and download in one click.

More agent tools will be exposed this way over time.
`,
	},
}

// authMD is the agent-facing authentication guide served at /auth.md.
// CreateMod.com is deliberately not an OAuth authorization server: API keys
// are provisioned by humans only, which keeps every credential traceable to
// an account the site can oversee and rate limit.
const authMD = `# Authentication on CreateMod.com

This document tells AI agents how authentication works on CreateMod.com
and what to do when an endpoint requires credentials.

## Summary

- Most agent-relevant functionality requires **no authentication** (see below).
- The JSON API (https://createmod.com/api) requires an **API key** sent as an
  ` + "`X-API-Key`" + ` header.
- There is **no autonomous agent registration**: no OAuth authorization
  server, no dynamic client registration, no token endpoint. API keys are
  created manually by a signed-in human.

## No authentication required

- MCP server: POST https://createmod.com/api/mcp (see
  /.well-known/mcp/server-card.json)
- Markdown content negotiation: request the home page or any schematic page
  with "Accept: text/markdown"
- Search suggestions: GET https://createmod.com/api/search/suggest?q={terms}
- Search and browse pages, sitemaps, and all /.well-known/ discovery files

## Getting an API key (requires a human)

1. Ask your user to create an account at https://createmod.com/register
   (or sign in at https://createmod.com/login).
2. The user creates an API key at https://createmod.com/settings/api-keys.
3. The user gives the key to you; send it as the ` + "`X-API-Key`" + ` header.

Agents must not create accounts or fill in registration forms on a user's
behalf.

## Rate limits

API keys default to 120 requests/minute; a different limit may be assigned
to individual keys. 429 responses include a Retry-After header. The
unauthenticated MCP endpoint is limited to 60 requests/minute per IP.

## Hard policy

Never download schematic (.nbt) files or call download endpoints, with or
without credentials. Schematic files are for players: send users to the
schematic page (https://createmod.com/schematics/{slug}) or to a generator
(https://createmod.com/generators/) to download builds themselves.
`

// AuthMDHandler serves /auth.md.
func AuthMDHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		e.Response.Header().Set("Cache-Control", "public, max-age=3600")
		return e.Blob(http.StatusOK, "text/markdown; charset=utf-8", []byte(authMD))
	}
}

// AgentSkillsIndexHandler serves /.well-known/agent-skills/index.json.
// mcpRegistryAuth is the domain-verification proof for publishing the
// com.createmod/* namespace to the official MCP registry
// (registry.modelcontextprotocol.io). The matching Ed25519 private key is
// held offline by the site owner; rotating the key means regenerating this
// public half and re-running mcp-publisher login http.
const mcpRegistryAuth = "v=MCPv1; k=ed25519; p=NgRkTZGoyo8EHwnqafyRGw5gfbXykQn+8EpBWj7D95A="

// MCPRegistryAuthHandler serves /.well-known/mcp-registry-auth.
func MCPRegistryAuthHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		e.Response.Header().Set("Cache-Control", "public, max-age=3600")
		return e.String(http.StatusOK, mcpRegistryAuth)
	}
}

func AgentSkillsIndexHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		type entry struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Digest      string `json:"digest"`
		}
		index := struct {
			Schema string  `json:"$schema"`
			Skills []entry `json:"skills"`
		}{
			Schema: "https://schemas.agentskills.io/discovery/0.2.0/schema.json",
		}
		for _, s := range agentSkills {
			sum := sha256.Sum256([]byte(s.Body))
			index.Skills = append(index.Skills, entry{
				Name:        s.Name,
				Type:        "skill-md",
				Description: s.Description,
				URL:         "/.well-known/agent-skills/" + s.Name + "/SKILL.md",
				Digest:      "sha256:" + hex.EncodeToString(sum[:]),
			})
		}
		body, err := json.Marshal(index)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to build index")
		}
		e.Response.Header().Set("Cache-Control", "public, max-age=3600")
		return e.Blob(http.StatusOK, "application/json; charset=utf-8", body)
	}
}

// AgentSkillHandler serves /.well-known/agent-skills/{name}/SKILL.md.
func AgentSkillHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		name := e.Request.PathValue("name")
		for _, s := range agentSkills {
			if s.Name == name {
				e.Response.Header().Set("Cache-Control", "public, max-age=3600")
				return e.Blob(http.StatusOK, "text/markdown; charset=utf-8", []byte(s.Body))
			}
		}
		return e.String(http.StatusNotFound, "skill not found")
	}
}
