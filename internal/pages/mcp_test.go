package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"createmod/internal/server"
)

func mcpRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e := server.NewRequestEvent(rec, req)
	if err := MCPHandler(nil, nil, nil, nil)(e); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	return rec
}

func Test_MCP_Initialize(t *testing.T) {
	rec := mcpRequest(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
			Instructions string `json:"instructions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.Result.ServerInfo.Name != mcpServerName || resp.Result.ServerInfo.Version == "" {
		t.Errorf("missing serverInfo, got %+v", resp.Result.ServerInfo)
	}
	if !strings.Contains(resp.Result.Instructions, "Never attempt to download schematic (.nbt) files") {
		t.Errorf("instructions must state the NBT policy")
	}
}

func Test_MCP_ToolsList(t *testing.T) {
	rec := mcpRequest(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	body := rec.Body.String()
	for _, want := range []string{"search_schematics", "get_schematic", "inputSchema", "never fetch .nbt"} {
		if !strings.Contains(body, want) {
			t.Errorf("tools/list missing %q", want)
		}
	}
	// No tool may expose file downloads
	if strings.Contains(strings.ToLower(body), "download_") {
		t.Errorf("no download tools may be exposed")
	}
}

func Test_MCP_Protocol_Edges(t *testing.T) {
	// Unknown method → JSON-RPC method-not-found
	rec := mcpRequest(t, `{"jsonrpc":"2.0","id":3,"method":"resources/list"}`)
	if !strings.Contains(rec.Body.String(), "-32601") {
		t.Errorf("expected method not found error, got %s", rec.Body.String())
	}
	// Notification (no id) → 202
	rec = mcpRequest(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if rec.Code != 202 {
		t.Errorf("notifications should return 202, got %d", rec.Code)
	}
	// Parse error
	rec = mcpRequest(t, `{not json`)
	if !strings.Contains(rec.Body.String(), "-32700") {
		t.Errorf("expected parse error, got %s", rec.Body.String())
	}
	// GET → 405
	req := httptest.NewRequest("GET", "/api/mcp", nil)
	rr := httptest.NewRecorder()
	_ = MCPHandler(nil, nil, nil, nil)(server.NewRequestEvent(rr, req))
	if rr.Code != 405 {
		t.Errorf("GET should be 405, got %d", rr.Code)
	}
}

func Test_MCP_ServerCard(t *testing.T) {
	req := httptest.NewRequest("GET", "/.well-known/mcp/server-card.json", nil)
	rec := httptest.NewRecorder()
	if err := MCPServerCardHandler()(server.NewRequestEvent(rec, req)); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var card map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	for _, key := range []string{"serverInfo", "transport", "capabilities", "version", "remotes", "$schema"} {
		if _, ok := card[key]; !ok {
			t.Errorf("server card missing %q", key)
		}
	}
	transport := card["transport"].(map[string]any)
	if transport["endpoint"] != "https://createmod.com/api/mcp" {
		t.Errorf("unexpected transport endpoint: %v", transport["endpoint"])
	}
}

func Test_AgentSkills_Index_And_Digests(t *testing.T) {
	req := httptest.NewRequest("GET", "/.well-known/agent-skills/index.json", nil)
	rec := httptest.NewRecorder()
	if err := AgentSkillsIndexHandler()(server.NewRequestEvent(rec, req)); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var index struct {
		Schema string `json:"$schema"`
		Skills []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			URL    string `json:"url"`
			Digest string `json:"digest"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &index); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if index.Schema != "https://schemas.agentskills.io/discovery/0.2.0/schema.json" {
		t.Errorf("wrong $schema: %s", index.Schema)
	}
	if len(index.Skills) == 0 {
		t.Fatalf("no skills in index")
	}

	for _, sk := range index.Skills {
		// Fetch the skill file and verify the digest matches
		sreq := httptest.NewRequest("GET", sk.URL, nil)
		sreq.SetPathValue("name", sk.Name)
		srec := httptest.NewRecorder()
		if err := AgentSkillHandler()(server.NewRequestEvent(srec, sreq)); err != nil {
			t.Fatalf("skill fetch error: %v", err)
		}
		if srec.Code != 200 {
			t.Errorf("skill %s not served: %d", sk.Name, srec.Code)
			continue
		}
		sum := sha256.Sum256(srec.Body.Bytes())
		if sk.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
			t.Errorf("digest mismatch for %s", sk.Name)
		}
		body := srec.Body.String()
		if !strings.HasPrefix(body, "---\nname: "+sk.Name) {
			t.Errorf("skill %s missing frontmatter", sk.Name)
		}
		if !strings.Contains(body, ".nbt") {
			t.Errorf("skill %s must state the NBT policy", sk.Name)
		}
	}

	// Unknown skill → 404
	nreq := httptest.NewRequest("GET", "/.well-known/agent-skills/nope/SKILL.md", nil)
	nreq.SetPathValue("name", "nope")
	nrec := httptest.NewRecorder()
	_ = AgentSkillHandler()(server.NewRequestEvent(nrec, nreq))
	if nrec.Code != 404 {
		t.Errorf("unknown skill should 404, got %d", nrec.Code)
	}
}

func Test_AuthMD(t *testing.T) {
	req := httptest.NewRequest("GET", "/auth.md", nil)
	rec := httptest.NewRecorder()
	if err := AuthMDHandler()(server.NewRequestEvent(rec, req)); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/markdown") {
		t.Errorf("expected text/markdown, got %s", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"X-API-Key",
		"/settings/api-keys",
		"no autonomous agent registration",
		"must not create accounts",
		"Never download schematic (.nbt) files",
		"120 requests/minute",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("auth.md missing %q", want)
		}
	}
}
