package pages

import "testing"

func Test_youtubeID_VariousForms(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ":             "dQw4w9WgXcQ",
		"https://youtube.com/watch?v=dQw4w9WgXcQ&ab_channel=Rick": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                            "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":              "dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=abcd_123-XYZ":            "abcd_123-XYZ",
		"v=dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"":              "",
		"invalid":       "",
	}
	for in, want := range cases {
		got := youtubeID(in)
		if got != want {
			t.Fatalf("youtubeID(%q) = %q; want %q", in, got, want)
		}
	}
}

func Test_youtubeThumb(t *testing.T) {
	if youtubeThumb("") != "" {
		t.Fatalf("youtubeThumb empty input should return empty string")
	}
	id := "dQw4w9WgXcQ"
	got := youtubeThumb(id)
	want := "https://i.ytimg.com/vi/" + id + "/mqdefault.jpg"
	if got != want {
		t.Fatalf("youtubeThumb(%q) = %q; want %q", id, got, want)
	}
}
