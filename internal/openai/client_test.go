package openai

import "testing"

func TestUpholdFromReviewAnswer(t *testing.T) {
	tests := []struct {
		name       string
		answer     string
		wantUphold bool
	}{
		{"plain ok clears flag", "ok", false},
		{"ok with punctuation clears flag", "OK.", false},
		{"ok quoted clears flag", "\"ok\"", false},
		{"ok with trailing words clears flag", "ok, this is harmless slang", false},
		{"violation upholds flag", "violation", true},
		{"violation with detail upholds flag", "violation - credible threat", true},
		{"empty answer upholds flag (fail safe)", "", true},
		{"unexpected answer upholds flag (fail safe)", "I am not sure", true},
		{"okay (not the token) upholds flag", "okay maybe", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := upholdFromReviewAnswer(tt.answer); got != tt.wantUphold {
				t.Errorf("upholdFromReviewAnswer(%q) = %v, want %v", tt.answer, got, tt.wantUphold)
			}
		})
	}
}

func TestIsAffirmativeTrue(t *testing.T) {
	// Guard the quality-check parser: approvals must be recognized and negatives
	// (rejections with a reason) must not be misread as approval.
	approvals := []string{"true", "True.", "\"true\"", "true, valid Minecraft build"}
	for _, s := range approvals {
		if !isAffirmativeTrue(s) {
			t.Errorf("isAffirmativeTrue(%q) = false, want true", s)
		}
	}
	rejections := []string{"not a Minecraft image", "false", "not true", "just random blocks", "spam/low effort"}
	for _, s := range rejections {
		if isAffirmativeTrue(s) {
			t.Errorf("isAffirmativeTrue(%q) = true, want false", s)
		}
	}
}
