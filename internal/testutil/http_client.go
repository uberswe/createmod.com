package testutil

import (
	"net/http"
	"net/http/cookiejar"
	"testing"
)

// NewHTTPClient returns an http.Client with a cookie jar for stateful requests in tests.
func NewHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// WithHTMX decorates a request with basic HTMX headers to simulate HX requests in tests.
func WithHTMX(req *http.Request) *http.Request {
	req.Header.Set("HX-Request", "true")
	return req
}
