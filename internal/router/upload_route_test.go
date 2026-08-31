package router

import "testing"

// TestIsUploadRoute guards the larger request-body cap for endpoints that carry
// an .nbt plus inline screenshots. Regression: /u/{token}/make-public was NOT
// covered, so publishing with a few MB of images hit the default 10 MB cap and
// returned a misleading "under 100 MB" error.
func TestIsUploadRoute(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/u/abc123def/make-public", true},
		{"/schematics/my-cool-build/update", true},
		{"/admin/schematics/abc123", true},
		{"/upload/nbt", true},
		{"/api/images/upload", true},
		{"/api/schematics/upload", true},
		{"/api/schematics/upload-anonymous", true},
		{"/u/abc123/add-file", true},
		// Not upload routes:
		{"/schematics/my-cool-build", false},
		{"/api/schematics/my-cool-build/download", false},
		{"/schematics", false},
		{"/", false},
		{"/api/notifications/unread-count", false},
	}
	for _, c := range cases {
		if got := isUploadRoute(c.path); got != c.want {
			t.Errorf("isUploadRoute(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
