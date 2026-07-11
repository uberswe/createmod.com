package pages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"createmod/internal/server"
)

func Test_NBTTreeUploadHandler(t *testing.T) {
	nbt := suspiciousNBT(t) // has a command block BE — good tree depth

	// Root tree
	body, ctype := multipartBody(t, nil, "file", "a.nbt", nbt)
	req := httptest.NewRequest(http.MethodPost, "/api/nbt-tree", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	if err := NBTTreeUploadHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var page struct {
		Node     map[string]interface{}   `json:"node"`
		Children []map[string]interface{} `json:"children"`
		Total    int                      `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total < 4 {
		t.Errorf("root total = %d", page.Total)
	}
	var blocksPath string
	for _, c := range page.Children {
		if c["name"] == "blocks" {
			blocksPath, _ = c["path"].(string)
		}
	}
	if blocksPath == "" {
		t.Fatalf("no blocks child: %+v", page.Children)
	}

	// SNBT of a subtree
	body, ctype = multipartBody(t, nil, "file", "a.nbt", nbt)
	req = httptest.NewRequest(http.MethodPost, "/api/nbt-tree?snbt=1&path="+blocksPath, body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := NBTTreeUploadHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	var snbtResp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &snbtResp)
	if !strings.Contains(snbtResp["snbt"], "pos") {
		t.Errorf("snbt = %.80s", snbtResp["snbt"])
	}

	// Key search finds the command
	body, ctype = multipartBody(t, nil, "file", "a.nbt", nbt)
	req = httptest.NewRequest(http.MethodPost, "/api/nbt-tree?q=Command", body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := NBTTreeUploadHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	var searchResp struct {
		Results []map[string]interface{} `json:"results"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &searchResp)
	if len(searchResp.Results) == 0 {
		t.Errorf("no search hits for Command: %s", rec.Body.String())
	}

	// Garbage rejected
	body, ctype = multipartBody(t, nil, "file", "junk.nbt", []byte("garbage"))
	req = httptest.NewRequest(http.MethodPost, "/api/nbt-tree", body)
	req.Header.Set("Content-Type", ctype)
	rec = httptest.NewRecorder()
	if err := NBTTreeUploadHandler()(&server.RequestEvent{Response: rec, Request: req}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("garbage: status %d", rec.Code)
	}
}

func Test_NBTViewer_Templates(t *testing.T) {
	viewer, err := os.ReadFile(filepath.Join("..", "..", "template", "nbt_viewer.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"nv-drop", "/api/nbt-tree", "nbt-tree.js", "NBTTree.mount"} {
		if !strings.Contains(string(viewer), m) {
			t.Errorf("nbt_viewer.html missing %q", m)
		}
	}
	sch, err := os.ReadFile(filepath.Join("..", "..", "template", "schematic.html"))
	if err != nil {
		t.Fatal(err)
	}
	// The schematic page links to the dedicated /nbt-data page.
	for _, m := range []string{"/nbt-data", "NBT Data"} {
		if !strings.Contains(string(sch), m) {
			t.Errorf("schematic.html missing %q", m)
		}
	}
	nd, err := os.ReadFile(filepath.Join("..", "..", "template", "nbt_data.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"NBTTree.mount", "/nbt-tree", "Back to schematic"} {
		if !strings.Contains(string(nd), m) {
			t.Errorf("nbt_data.html missing %q", m)
		}
	}
	js, err := os.ReadFile(filepath.Join("..", "..", "template", "static", "nbt-tree.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"window.NBTTree", "load more", "copy path", "SNBT"} {
		if !strings.Contains(string(js), m) {
			t.Errorf("nbt-tree.js missing %q", m)
		}
	}
}
