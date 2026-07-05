package pages

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"createmod/internal/store"
)

func structHasField(v interface{}, name string) bool {
	_, ok := reflect.TypeOf(v).FieldByName(name)
	return ok
}

func Test_EffectiveRateLimit(t *testing.T) {
	cases := []struct {
		name string
		key  *store.APIKey
		def  int
		want int
	}{
		{"nil key uses default", nil, 120, 120},
		{"zero override uses default", &store.APIKey{}, 120, 120},
		{"custom limit wins", &store.APIKey{RateLimitPerMinute: 1000}, 120, 1000},
		{"custom below default wins", &store.APIKey{RateLimitPerMinute: 5}, 120, 5},
		{"negative treated as unset", &store.APIKey{RateLimitPerMinute: -1}, 120, 120},
	}
	for _, c := range cases {
		if got := effectiveRateLimit(c.key, c.def); got != c.want {
			t.Errorf("%s: effectiveRateLimit = %d, want %d", c.name, got, c.want)
		}
	}
}

func renderAdminAPIKeys(t *testing.T, d AdminAPIKeysData) string {
	t.Helper()
	r := NewTestRegistry()

	root := projectRootFromThisFile(t)
	var paths []string
	for _, f := range adminAPIKeysTemplates {
		paths = append(paths, filepath.Join(root, f))
	}

	d.DefaultData.Language = "en"
	out, err := r.LoadFiles(paths...).Render(d)
	if err != nil {
		t.Fatalf("render admin_api_keys.html failed: %v", err)
	}
	return out
}

func Test_Admin_API_Keys_Template_Lists_Keys(t *testing.T) {
	d := AdminAPIKeysData{
		DefaultRateLimit: 120,
		Keys: []AdminAPIKeyRow{
			{
				ID: "k1", Username: "alice", Label: "prod bot", Last8: "abcd1234",
				Created: "2026-01-01 10:00", LastUsed: "2026-07-01 12:00",
				Usage24h: 42, Usage7d: 300, UsageTotal: 9000, RateLimit: 1000,
				Endpoints: []AdminAPIKeyEndpointRow{
					{Endpoint: "GET /api/schematics", Requests: 250, LastUsed: "2026-07-01 12:00"},
				},
			},
			{
				ID: "k2", Username: "bob", Last8: "efgh5678",
				Created: "2026-02-01 10:00",
			},
		},
	}
	out := renderAdminAPIKeys(t, d)

	for _, want := range []string{
		"alice", "prod bot", "abcd1234", "GET /api/schematics",
		`value="1000"`, "custom",
		"bob", "efgh5678", "never",
		"/admin/api-keys/k1/rate-limit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected rendered page to contain %q", want)
		}
	}
	// Key without an override shows the placeholder, not a value
	if !strings.Contains(out, `placeholder="default"`) {
		t.Errorf("expected default placeholder on rate limit input")
	}
}

func Test_Admin_API_Keys_Row_Carries_No_Secret_Key_Material(t *testing.T) {
	// Keys are secrets: the admin view model may expose the last8 display
	// fragment (same as the user's own settings page) but never the hash
	// or full key.
	fields := []string{"KeyHash", "Secret", "Key"}
	row := AdminAPIKeyRow{}
	for _, f := range fields {
		if structHasField(row, f) {
			t.Errorf("AdminAPIKeyRow must not have field %q", f)
		}
	}
}

func Test_FilterAdminAPIKeyRows(t *testing.T) {
	rows := []AdminAPIKeyRow{
		{ID: "k1", Username: "Alice", Label: "prod bot"},
		{ID: "k2", Username: "bob", Label: ""},
		{ID: "k3", Username: "carol", Label: "alice-integration"},
	}
	if got := filterAdminAPIKeyRows(rows, ""); len(got) != 3 {
		t.Errorf("empty query should return all rows, got %d", len(got))
	}
	if got := filterAdminAPIKeyRows(rows, "ALICE"); len(got) != 2 {
		t.Errorf("case-insensitive username/label match should return 2 rows, got %d", len(got))
	}
	if got := filterAdminAPIKeyRows(rows, "bob"); len(got) != 1 || got[0].ID != "k2" {
		t.Errorf("expected only bob's key, got %+v", got)
	}
	if got := filterAdminAPIKeyRows(rows, "nomatch"); len(got) != 0 {
		t.Errorf("expected no rows, got %d", len(got))
	}
}

func Test_Admin_API_Keys_Template_Empty_State(t *testing.T) {
	out := renderAdminAPIKeys(t, AdminAPIKeysData{DefaultRateLimit: 120})
	if !strings.Contains(out, "No API keys have been created yet") {
		t.Errorf("expected empty state message")
	}
}
