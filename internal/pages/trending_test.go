package pages

import (
	"math"
	"testing"
	"time"
)

func Test_TrendingScore_NewerBeatsOlder_WithSimilarEngagement(t *testing.T) {
	createdOld := time.Date(2025, 10, 6, 8, 0, 0, 0, time.UTC)  // 7 days before "now"
	createdNew := time.Date(2025, 10, 12, 8, 0, 0, 0, time.UTC) // 1 day before "now"

	recentViews := 100.0
	totalViews := 500.0
	ratingCount := 5.0
	ratingSum := 20.0

	sOld := trendingScore(createdOld, recentViews, totalViews, ratingCount, ratingSum, 0, 0)
	sNew := trendingScore(createdNew, recentViews, totalViews, ratingCount, ratingSum, 0, 0)

	if !(sNew > sOld) {
		t.Fatalf("expected newer item to score higher than older with same engagement: new=%f old=%f", sNew, sOld)
	}
}

func Test_TrendingScore_EngagementCanOvercomeAge(t *testing.T) {
	createdOld := time.Date(2025, 10, 11, 8, 0, 0, 0, time.UTC) // 2 days before "now"
	createdNew := time.Date(2025, 10, 13, 2, 0, 0, 0, time.UTC) // 6 hours before "now"

	// Old item has substantially higher engagement
	sOld := trendingScore(createdOld, 2000, 50000, 100, 400, 500, 10000)
	sNew := trendingScore(createdNew, 5, 10, 1, 3, 0, 0)

	if !(sOld > sNew) {
		t.Fatalf("expected much higher engagement to overcome age penalty: old=%f new=%f", sOld, sNew)
	}
}

func Test_TrendingScore_MonotonicWithSignals(t *testing.T) {
	created := time.Date(2025, 10, 12, 20, 0, 0, 0, time.UTC)

	base := trendingScore(created, 10, 100, 2, 8, 5, 50)
	moreRecentViews := trendingScore(created, 50, 100, 2, 8, 5, 50)
	moreTotalViews := trendingScore(created, 10, 1000, 2, 8, 5, 50)
	moreRatingCount := trendingScore(created, 10, 100, 10, 8, 5, 50)
	moreRatingSum := trendingScore(created, 10, 100, 2, 40, 5, 50)
	moreRecentDownloads := trendingScore(created, 10, 100, 2, 8, 50, 50)
	moreTotalDownloads := trendingScore(created, 10, 100, 2, 8, 5, 500)

	if !(moreRecentViews > base) {
		t.Fatalf("expected score to increase with more recent views: base=%f moreRecentViews=%f", base, moreRecentViews)
	}
	if !(moreTotalViews > base) {
		t.Fatalf("expected score to increase with more total views: base=%f moreTotalViews=%f", base, moreTotalViews)
	}
	if !(moreRatingCount > base) {
		t.Fatalf("expected score to increase with more rating count: base=%f moreRatingCount=%f", base, moreRatingCount)
	}
	if !(moreRatingSum > base) {
		t.Fatalf("expected score to increase with more rating sum: base=%f moreRatingSum=%f", base, moreRatingSum)
	}
	if !(moreRecentDownloads > base) {
		t.Fatalf("expected score to increase with more recent downloads: base=%f moreRecentDownloads=%f", base, moreRecentDownloads)
	}
	if !(moreTotalDownloads > base) {
		t.Fatalf("expected score to increase with more total downloads: base=%f moreTotalDownloads=%f", base, moreTotalDownloads)
	}
}

func Test_TrendingScore_ZeroEngagement_SortsByNewest(t *testing.T) {
	older := time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 10, 12, 0, 0, 0, 0, time.UTC)

	sOlder := trendingScore(older, 0, 0, 0, 0, 0, 0)
	sNewer := trendingScore(newer, 0, 0, 0, 0, 0, 0)

	// With zero engagement, log10(1) = 0, so score is purely based on creation time
	if !(sNewer > sOlder) {
		t.Fatalf("with zero engagement, newer should rank higher: newer=%f older=%f", sNewer, sOlder)
	}
}

func Test_TrendingScore_Finite(t *testing.T) {
	created := time.Now().UTC()
	s := trendingScore(created, 0, 0, 0, 0, 0, 0)
	if math.IsNaN(s) || math.IsInf(s, 0) {
		t.Fatalf("score should be finite, got %v", s)
	}
}

func Test_TrendingScore_75DayTimescale(t *testing.T) {
	// An item from 75 days ago needs ~10x engagement to match a new item
	now := time.Date(2025, 10, 13, 0, 0, 0, 0, time.UTC)
	timescaleAgo := now.Add(-75 * 24 * time.Hour)

	// New item with 10 engagement units: log10(10) = 1
	sNew := trendingScore(now, 10, 0, 0, 0, 0, 0)
	// Old item: needs ~100 engagement to get log10(100) = 2, compensating for 1 timescale period
	sOld := trendingScore(timescaleAgo, 100, 0, 0, 0, 0, 0)

	// They should be approximately equal (within 0.15)
	diff := math.Abs(sNew - sOld)
	if diff > 0.15 {
		t.Fatalf("75-day-old item with 10x engagement should roughly match new item: new=%f old=%f diff=%f", sNew, sOld, diff)
	}
}
