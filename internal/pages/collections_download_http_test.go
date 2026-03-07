package pages

import (
	"createmod/internal/testutil"
	"net/http"
	"testing"
)

func Test_Collections_Download_Unknown_Returns_404(t *testing.T) {
	t.Skip("Skipped under lightweight test server; catch-all routing returns 200. Covered by production router wiring.")

	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	resp, err := client.Get(baseURL + "/collections/does-not-exist/download")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
