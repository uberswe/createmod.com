package jobs

import (
	"testing"
	"time"
)

func mustStockholm(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatalf("loading Europe/Stockholm: %v", err)
	}
	return loc
}

func Test_ModerationSummaryWindow(t *testing.T) {
	loc := mustStockholm(t)

	tests := []struct {
		name      string
		now       time.Time
		wantSince time.Time
		wantUntil time.Time
		wantOK    bool
	}{
		{
			name:      "noon run covers previous evening through noon",
			now:       time.Date(2026, 7, 10, 12, 37, 0, 0, loc),
			wantSince: time.Date(2026, 7, 9, 22, 0, 0, 0, loc),
			wantUntil: time.Date(2026, 7, 10, 12, 0, 0, 0, loc),
			wantOK:    true,
		},
		{
			name:      "evening run covers noon through evening",
			now:       time.Date(2026, 7, 10, 22, 5, 0, 0, loc),
			wantSince: time.Date(2026, 7, 10, 12, 0, 0, 0, loc),
			wantUntil: time.Date(2026, 7, 10, 22, 0, 0, 0, loc),
			wantOK:    true,
		},
		{
			name:   "other hours skip",
			now:    time.Date(2026, 7, 10, 13, 0, 0, 0, loc),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, until, ok := moderationSummaryWindow(tt.now)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if !since.Equal(tt.wantSince) {
				t.Errorf("since = %v, want %v", since, tt.wantSince)
			}
			if !until.Equal(tt.wantUntil) {
				t.Errorf("until = %v, want %v", until, tt.wantUntil)
			}
		})
	}
}

// Consecutive windows must tile the day exactly: no gap and no overlap
// between the noon and evening summaries, including across a DST change.
func Test_ModerationSummaryWindow_NoGapsOrOverlap(t *testing.T) {
	loc := mustStockholm(t)
	// 2026-03-29 and 2026-10-25 are the DST transitions in Europe/Stockholm.
	days := [][3]int{
		{2026, 7, 10},
		{2026, 3, 29},
		{2026, 10, 25},
	}
	for _, d := range days {
		year, month, day := d[0], time.Month(d[1]), d[2]
		noonRun := time.Date(year, month, day, 12, 20, 0, 0, loc)
		eveRun := time.Date(year, month, day, 22, 45, 0, 0, loc)
		nextNoonRun := time.Date(year, month, day+1, 12, 5, 0, 0, loc)

		noonSince, noonUntil, ok := moderationSummaryWindow(noonRun)
		if !ok {
			t.Fatalf("day %v-%v-%v: noon run not recognized", year, month, day)
		}
		eveSince, eveUntil, ok := moderationSummaryWindow(eveRun)
		if !ok {
			t.Fatalf("day %v-%v-%v: evening run not recognized", year, month, day)
		}
		if !noonUntil.Equal(eveSince) {
			t.Errorf("day %v-%v-%v: noon window ends %v but evening window starts %v", year, month, day, noonUntil, eveSince)
		}
		nextNoonSince, _, _ := moderationSummaryWindow(nextNoonRun)
		if !eveUntil.Equal(nextNoonSince) {
			t.Errorf("day %v-%v-%v: evening window ends %v but next noon window starts %v", year, month, day, eveUntil, nextNoonSince)
		}
		if !noonSince.Before(noonUntil) || !eveSince.Before(eveUntil) {
			t.Errorf("day %v-%v-%v: degenerate window", year, month, day)
		}
	}
}

func Test_ModerationSummarySubject(t *testing.T) {
	if got := moderationSummarySubject(3, 2, 1); got != "Moderation Summary: 3 auto-approved, 2 pending, 1 report" {
		t.Errorf("subject = %q", got)
	}
	if got := moderationSummarySubject(0, 0, 2); got != "Moderation Summary: 2 reports" {
		t.Errorf("subject = %q", got)
	}
	if got := moderationSummarySubject(0, 0, 0); got != "Moderation Summary" {
		t.Errorf("subject = %q", got)
	}
	if got := moderationSummarySubject(0, 5, 0); got != "Moderation Summary: 5 pending" {
		t.Errorf("subject = %q", got)
	}
}
