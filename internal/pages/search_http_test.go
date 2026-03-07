package pages

import (
	"createmod/internal/testutil"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_Search_HTTP_HTMX_LoggedOut(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	form := url.Values{}
	form.Set("advanced-search-term", "gears")

	req, err := http.NewRequest("POST", baseURL+"/search", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req = testutil.WithHTMX(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Read body
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(b)

	if !strings.Contains(body, "/login") {
		t.Errorf("expected logged-out response to contain /login link")
	}
	if strings.Contains(body, "/logout") {
		t.Errorf("expected logged-out response not to contain /logout link")
	}
}

func Test_Search_HTTP_HTMX_LoggedIn(t *testing.T) {
	baseURL, cleanup := testutil.NewTestServer(t)
	defer cleanup()

	client := testutil.NewHTTPClient(t)

	// Set an auth cookie to simulate logged-in state
	// Any non-empty value should be treated as authenticated by the test server.
	cookie := &http.Cookie{Name: "create-mod-auth", Value: "test-token", Path: "/"}
	req0, _ := http.NewRequest("GET", baseURL+"/", nil)
	client.Jar.SetCookies(req0.URL, []*http.Cookie{cookie})

	form := url.Values{}
	form.Set("advanced-search-term", "gears")

	req, err := http.NewRequest("POST", baseURL+"/search", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req = testutil.WithHTMX(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	b2, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(b2)

	if !strings.Contains(body, "/logout") {
		t.Errorf("expected logged-in response to contain /logout link")
	}
	if !strings.Contains(body, "/profile") {
		t.Errorf("expected logged-in response to contain /profile link")
	}
	if strings.Contains(body, "/login") {
		t.Errorf("expected logged-in response not to contain /login link")
	}
}
