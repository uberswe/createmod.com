package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_AdminDashboard_Template_Has_Expected_Elements(t *testing.T) {
	path := filepath.Join("..", "..", "template", "admin_dashboard.html")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	must := []string{
		"Admin Dashboard",
		"Pending Schematics",
		"Open Reports",
		"Pending Categories",
		"Pending Tags",
		"Total Schematics",
		"Moderated",
		"Deleted",
		"Recent Pending Schematics",
		"View All Pending",
		"/admin/schematics?filter=pending",
		"/admin/reports",
		"/admin/tags",
	}
	for _, m := range must {
		if !strings.Contains(s, m) {
			t.Fatalf("admin_dashboard.html missing: %s", m)
		}
	}
}
