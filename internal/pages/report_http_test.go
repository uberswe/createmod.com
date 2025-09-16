package pages

import (
	"createmod/internal/testutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_Reports_HTTP_NormalRedirect(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

	form := url.Values{}
	form.Set("target_type", "schematic")
	form.Set("target_id", "abc123")
	form.Set("reason", "spam")
	form.Set("return_to", "/schematics/abc123")

	req, _ := http.NewRequest("POST", baseURL+"/reports", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/schematics/abc123" {
		t.Fatalf("expected redirect to /schematics/abc123, got %q", loc)
	}
}

func Test_Reports_HTTP_HTMX_Redirect(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	form := url.Values{}
	form.Set("target_type", "comment")
	form.Set("target_id", "cmt1")
	form.Set("reason", "abuse")
	form.Set("return_to", "/schematics/gearbox#cmt1")

	req, _ := http.NewRequest("POST", baseURL+"/reports", strings.NewReader(form.Encode()))
	req = testutil.WithHTMX(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}
	if hdr := resp.Header.Get("HX-Redirect"); hdr != "/schematics/gearbox#cmt1" {
		t.Fatalf("expected HX-Redirect '/schematics/gearbox#cmt1', got %q", hdr)
	}
}
