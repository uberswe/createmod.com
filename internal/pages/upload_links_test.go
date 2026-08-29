package pages

import "testing"

func TestDescriptionHasReviewableLinks(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"no links", "A lovely steam train build with passenger cars.", false},
		{"reddit allowed", "Discussion: https://www.reddit.com/r/CreateMod/comments/abc", false},
		{"reddit short domain", "See https://redd.it/abc123", false},
		{"youtube allowed", "Showcase: https://youtu.be/dQw4w9WgXcQ", false},
		{"youtube full allowed", "https://www.youtube.com/watch?v=abc", false},
		{"on-site allowed", "More at https://createmod.com/schematics/foo", false},
		{"discord flagged", "Join https://discord.gg/abcdef", true},
		{"google drive flagged", "Download https://drive.google.com/file/d/xyz/view", true},
		{"mega flagged", "Mirror: https://mega.co.nz/file/abc", true},
		{"mega new domain flagged", "https://mega.nz/file/abc", true},
		{"reddit plus discord flagged", "reddit https://reddit.com/r/x and https://discord.gg/y", true},
		{"http scheme flagged", "http://example.com/thing", true},
		{"case-insensitive host allowed", "HTTPS://WWW.REDDIT.COM/r/x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := descriptionHasReviewableLinks(c.text); got != c.want {
				t.Errorf("descriptionHasReviewableLinks(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}
