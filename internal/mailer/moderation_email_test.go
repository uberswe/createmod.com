package mailer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModerationEmailsRender checks each moderation email renders, contains its
// key copy, and never leaves an unsubstituted format verb. When CM_FIXTURE_DIR
// is set it also writes the HTML for a browser eyeball.
func TestModerationEmailsRender(t *testing.T) {
	out := os.Getenv("CM_FIXTURE_DIR")
	cases := []struct {
		file string
		html string
		want []string
	}{
		{"1-live", SchematicLiveEmail("Steam Train", "https://createmod.com/schematics/steam-train"),
			[]string{"Steam Train is live", "View your schematic", "#2f9e44"}},
		{"2-action", SchematicActionNeededEmail("Windmill", "https://createmod.com/schematics/windmill", []string{"Add more detail.", "Add the Create version."}),
			[]string{"Action needed", "Add more detail.", "Add the Create version.", "Fix it now", "No deadline"}},
		{"3-image", SchematicImageReviewEmail("Airship", "https://createmod.com/schematics/airship", 48),
			[]string{"being reviewed", "48 hours", "Shaders sometimes", "upload a replacement"}},
		{"4a-reject-fixable", SchematicNotPublishedEmail("Odd", "https://createmod.com/schematics/odd", "Bad words.", true),
			[]string{"was not published", "resubmit once", "Bad words."}},
		{"4b-reject-final", SchematicNotPublishedEmail("Bad", "https://createmod.com/schematics/bad", "Not a build.", false),
			[]string{"was not published", "A human confirmed", "Not a build."}},
	}
	for _, c := range cases {
		if strings.Contains(c.html, "%!") {
			t.Errorf("%s: unsubstituted format verb in output", c.file)
		}
		for _, w := range c.want {
			if !strings.Contains(c.html, w) {
				t.Errorf("%s: missing %q", c.file, w)
			}
		}
		if out != "" {
			_ = os.WriteFile(filepath.Join(out, c.file+".html"), []byte(c.html), 0o644)
		}
	}
}

// TestModerationEmailEscaping ensures author-controlled fields are escaped.
func TestModerationEmailEscaping(t *testing.T) {
	html := SchematicLiveEmail("<script>alert(1)</script>", "https://createmod.com/x")
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Error("title not escaped — XSS via email title")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("expected escaped title")
	}
}
