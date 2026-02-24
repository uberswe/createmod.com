package pages

import (
    "math"
    "testing"
    "time"
)

func Test_TrendingScore_NewerBeatsOlder_WithSimilarEngagement(t *testing.T) {
    now := time.Date(2025, 10, 13, 8, 0, 0, 0, time.UTC)
    createdOld := now.Add(-7 * 24 * time.Hour)  // 7 days ago
    createdNew := now.Add(-24 * time.Hour)      // 1 day ago

    views := 100.0
    ratingsSum := 10.0
    decay := 1.8
    ratingWeight := 2.0

    sOld := trendingScore(now, createdOld, views, ratingsSum, decay, ratingWeight)
    sNew := trendingScore(now, createdNew, views, ratingsSum, decay, ratingWeight)

    if !(sNew > sOld) {
        t.Fatalf("expected newer item to score higher than older with same engagement: new=%f old=%f", sNew, sOld)
    }
}

func Test_TrendingScore_EngagementCanOvercomeAge(t *testing.T) {
    now := time.Date(2025, 10, 13, 8, 0, 0, 0, time.UTC)
    createdOld := now.Add(-48 * time.Hour) // 2 days ago
    createdNew := now.Add(-6 * time.Hour)  // 6 hours ago

    // Old item has substantially higher engagement
    viewsOld := 2000.0
    ratingsOld := 200.0

    viewsNew := 50.0
    ratingsNew := 5.0

    decay := 1.8
    ratingWeight := 2.0

    sOld := trendingScore(now, createdOld, viewsOld, ratingsOld, decay, ratingWeight)
    sNew := trendingScore(now, createdNew, viewsNew, ratingsNew, decay, ratingWeight)

    if !(sOld > sNew) {
        t.Fatalf("expected much higher engagement to overcome age penalty: old=%f new=%f", sOld, sNew)
    }
}

func Test_TrendingScore_MonotonicWithSignals(t *testing.T) {
    now := time.Now().UTC()
    created := now.Add(-12 * time.Hour)
    decay := 1.8
    ratingWeight := 2.0

    base := trendingScore(now, created, 10, 1, decay, ratingWeight)
    moreViews := trendingScore(now, created, 20, 1, decay, ratingWeight)
    moreRatings := trendingScore(now, created, 10, 5, decay, ratingWeight)

    if !(moreViews > base) {
        t.Fatalf("expected score to increase with more views: base=%f moreViews=%f", base, moreViews)
    }
    if !(moreRatings > base) {
        t.Fatalf("expected score to increase with more ratings: base=%f moreRatings=%f", base, moreRatings)
    }

    // sanity: denominator never zero and finite
    s := trendingScore(now, created, 0, 0, decay, ratingWeight)
    if math.IsNaN(s) || math.IsInf(s, 0) {
        t.Fatalf("score should be finite, got %v", s)
    }
}
