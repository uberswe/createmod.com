package pages

import (
	"createmod/internal/testutil"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func Test_MakePublic_HTTP_Redirect(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// 1) Upload a temp NBT to get a token/preview URL
	ctype, body := buildMultipartNBTPayload(t, "nbt", "makepublic.nbt", "mpdata")

	upReq, err := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	upReq = testutil.WithHTMX(upReq)
	upReq.Header.Set("Content-Type", ctype)

	upResp, err := client.Do(upReq)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	defer upResp.Body.Close()

	if upResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on upload, got %d", upResp.StatusCode)
	}

	b, _ := io.ReadAll(upResp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("expected JSON response, got: %q", string(b))
	}
	previewURL, _ := result["url"].(string)
	if previewURL == "" {
		t.Fatalf("missing url in JSON response: %q", string(b))
	}

	// 2) POST make-public (non-HTMX) and expect 303 redirect to /upload/pending
	// Use a client that doesn't follow redirects to inspect Location
	noFollow := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

	mpReq, err := http.NewRequest("POST", baseURL+previewURL+"/make-public", nil)
	if err != nil {
		t.Fatalf("build make-public request: %v", err)
	}
	// note: no HTMX header here

	mpResp, err := noFollow.Do(mpReq)
	if err != nil {
		t.Fatalf("make-public request failed: %v", err)
	}
	defer mpResp.Body.Close()

	if mpResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", mpResp.StatusCode)
	}
	if loc := mpResp.Header.Get("Location"); loc != "/upload/pending" {
		t.Fatalf("expected redirect to /upload/pending, got %q", loc)
	}
}

func Test_MakePublic_HTTP_HTMX_Redirect(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// Upload to get preview URL
	ctype, body := buildMultipartNBTPayload(t, "nbt", "makepublic2.nbt", "data")
	upReq, err := http.NewRequest("POST", baseURL+"/upload/nbt", body)
	if err != nil {
		t.Fatalf("build upload request: %v", err)
	}
	upReq.Header.Set("Content-Type", ctype)
	upReq = testutil.WithHTMX(upReq)

	upResp, err := client.Do(upReq)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	b, _ := io.ReadAll(upResp.Body)
	_ = upResp.Body.Close()
	if upResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", upResp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("expected JSON response, got: %q", string(b))
	}
	previewURL, _ := result["url"].(string)
	if previewURL == "" {
		t.Fatalf("missing url in JSON response: %q", string(b))
	}

	// HTMX make-public expects 204 + HX-Redirect
	mpReq, err := http.NewRequest("POST", baseURL+previewURL+"/make-public", nil)
	if err != nil {
		t.Fatalf("build make-public request: %v", err)
	}
	mpReq = testutil.WithHTMX(mpReq)

	mpResp, err := client.Do(mpReq)
	if err != nil {
		t.Fatalf("make-public failed: %v", err)
	}
	defer mpResp.Body.Close()

	if mpResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", mpResp.StatusCode)
	}
	if hdr := mpResp.Header.Get("HX-Redirect"); hdr != "/upload/pending" {
		t.Fatalf("expected HX-Redirect '/upload/pending', got %q", hdr)
	}
}
