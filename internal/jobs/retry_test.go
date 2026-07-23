package jobs

import (
	"testing"
	"time"
)

func Test_slowRetryAt(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 30 * time.Minute},
		{2, 1 * time.Hour},
		{3, 2 * time.Hour},
		{4, 4 * time.Hour},
		{5, 8 * time.Hour},
		{6, 16 * time.Hour},
		{7, 24 * time.Hour},
		{8, 24 * time.Hour},
		{20, 24 * time.Hour},
	}
	for _, c := range cases {
		got := slowRetryAt(c.attempt, now).Sub(now)
		if got != c.want {
			t.Errorf("attempt %d: got delay %v, want %v", c.attempt, got, c.want)
		}
	}
}
