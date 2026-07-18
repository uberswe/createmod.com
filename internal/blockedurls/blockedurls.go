// Package blockedurls normalizes and matches admin-managed blocked URLs
// (e.g. DMCA takedown targets) against incoming request URLs.
package blockedurls

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Normalize converts an admin-supplied value (a full URL or an absolute path)
// into the canonical stored form: path plus optional query string, e.g.
// "/es/search?p=17&q=term". Scheme and host are discarded.
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if !strings.HasPrefix(u.Path, "/") {
		return "", fmt.Errorf("URL must be a full URL (https://...) or an absolute path starting with /")
	}
	normalized := u.EscapedPath()
	if u.RawQuery != "" {
		normalized += "?" + u.RawQuery
	}
	return normalized, nil
}

// Matches reports whether reqURL matches any of the stored entries. An entry
// matches when its path equals the request path and its query parameters
// equal the request's query parameters (order-insensitive).
func Matches(entries []string, reqURL *url.URL) bool {
	if reqURL == nil {
		return false
	}
	reqQuery := reqURL.Query()
	for _, entry := range entries {
		eu, err := url.Parse(entry)
		if err != nil {
			continue
		}
		if eu.Path != reqURL.Path {
			continue
		}
		if queryValuesEqual(eu.Query(), reqQuery) {
			return true
		}
	}
	return false
}

// queryValuesEqual reports whether two query parameter sets contain the same
// keys and values, ignoring parameter order.
func queryValuesEqual(a, b url.Values) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		bv, ok := b[key]
		if !ok || len(av) != len(bv) {
			return false
		}
		as := append([]string(nil), av...)
		bs := append([]string(nil), bv...)
		sort.Strings(as)
		sort.Strings(bs)
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
	}
	return true
}
