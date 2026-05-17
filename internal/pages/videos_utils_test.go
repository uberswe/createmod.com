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

func Test_IsValidYouTubeVideo(t *testing.T) {
	valid := []string{
		"",
		"dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtube.com/watch?v=dQw4w9WgXcQ&ab_channel=Rick",
		"https://m.youtube.com/watch?v=TMSfuN4pskY",
		"https://www.youtube.com/watch?app=desktop&v=ko7s8xiaYsU",
		"https://www.youtube.com/embed/0cn65dcAVs0",
		"https://www.youtube.com/shorts/599jzKlUJmk",
		"https://youtube.com/shorts/CKPLrg5RRM8",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://youtu.be/1PYVqA2j8Qo?si=BaMUFlUx9j_iyEcA",
		"https://www.youtube.com/watch?v=-FxPCZU7iiU&t=1s",
		"https://www.youtube.com/watch?v=abcd_123-XYZ",
		"-4yx-2s51P4",
	}
	for _, v := range valid {
		if !IsValidYouTubeVideo(v) {
			t.Errorf("IsValidYouTubeVideo(%q) = false; want true", v)
		}
	}

	invalid := []string{
		"https://www.reddit.com/r/CreateMod/comments/123/test/",
		"https://www.tiktok.com/@user/video/123",
		"https://vimeo.com/123456",
		"https://drive.google.com/file/d/abc/view",
		"https://imgur.com/a/8hKDfab",
		"https://www.youtube.com/channel/UC3Wy0ZIiTrF03v3wppMckjQ",
		"https://www.youtube.com/clip/UgkxIL6Q4OsIxvBPk8J_Uv1Nv-AslhqCWDYN",
		"https://Youtube",
		"No",
		"there is no vid",
		"awsfse",
		"[",
		"toolongforavideoid",
		"short",
	}
	for _, v := range invalid {
		if IsValidYouTubeVideo(v) {
			t.Errorf("IsValidYouTubeVideo(%q) = true; want false", v)
		}
	}
}

func Test_youtubeThumb(t *testing.T) {
	if youtubeThumb("") != "" {
		t.Fatalf("youtubeThumb empty input should return empty string")
	}
	id := "dQw4w9WgXcQ"
	got := youtubeThumb(id)
	want := "https://i.ytimg.com/vi/" + id + "/hq720.jpg"
	if got != want {
		t.Fatalf("youtubeThumb(%q) = %q; want %q", id, got, want)
	}
}
