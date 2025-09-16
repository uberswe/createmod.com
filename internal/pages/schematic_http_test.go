package pages

import (
	"createmod/internal/testutil"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func Test_Schematic_ViewCounter_Increments(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	name := "gearbox"

	// First view
	resp1, err := client.Get(baseURL + "/schematics/" + name)
	if err != nil {
		t.Fatalf("first view request failed: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on first view, got %d", resp1.StatusCode)
	}

	// Second view
	resp2, err := client.Get(baseURL + "/schematics/" + name)
	if err != nil {
		t.Fatalf("second view request failed: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on second view, got %d", resp2.StatusCode)
	}

	// Check stats
	statResp, err := client.Get(baseURL + "/_stats/views/" + name)
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	defer statResp.Body.Close()

	if statResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on stats, got %d", statResp.StatusCode)
	}
	b, _ := io.ReadAll(statResp.Body)
	gotStr := strings.TrimSpace(string(b))
	got, err := strconv.Atoi(gotStr)
	if err != nil {
		t.Fatalf("failed to parse views count %q: %v", gotStr, err)
	}
	if got != 2 {
		t.Fatalf("expected views=2, got %d", got)
	}
}

func Test_Schematic_DownloadCounter_Increments(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	name := "gearbox"

	// Trigger one download
	dresp, err := client.Get(baseURL + "/download/" + name)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	_ = dresp.Body.Close()
	if dresp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on download, got %d", dresp.StatusCode)
	}

	// Check stats
	statResp, err := client.Get(baseURL + "/_stats/downloads/" + name)
	if err != nil {
		t.Fatalf("stats request failed: %v", err)
	}
	defer statResp.Body.Close()

	if statResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on stats, got %d", statResp.StatusCode)
	}
	b, _ := io.ReadAll(statResp.Body)
	gotStr := strings.TrimSpace(string(b))
	got, err := strconv.Atoi(gotStr)
	if err != nil {
		t.Fatalf("failed to parse downloads count %q: %v", gotStr, err)
	}
	if got != 1 {
		t.Fatalf("expected downloads=1, got %d", got)
	}
}
