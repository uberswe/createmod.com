package ratelimit

import (
	"context"
	"time"
)

// PaidAIPerHour is the per-user hourly budget for money-costing OpenAI calls
// (quality checks, translation, image-build checks). The free omni-moderation
// safety checks are NOT counted against it. Callers that exceed the budget
// should DELAY the work (e.g. River JobSnooze), not drop it. (#1646)
const PaidAIPerHour = 10

// paidAIWindow is the budget window.
const paidAIWindow = time.Hour

// AllowPaidAI reserves one unit of a user's hourly paid-AI budget and reports
// whether the call is within budget. An empty userID (anonymous/system) or a nil
// limiter is always allowed — the budget is a per-authenticated-user cost cap,
// and we fail open so a limiter outage never blocks real work.
func AllowPaidAI(ctx context.Context, rl Limiter, userID string) bool {
	if rl == nil || userID == "" {
		return true
	}
	ok, _ := rl.Allow(ctx, "aicost:"+userID, PaidAIPerHour, paidAIWindow)
	return ok
}
