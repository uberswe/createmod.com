package ratelimit

import (
	"context"
	"testing"
)

func TestAllowPaidAI(t *testing.T) {
	ctx := context.Background()
	rl := NewMemory()
	defer rl.Close()

	// First PaidAIPerHour calls for a user are allowed; the next is denied.
	for i := 0; i < PaidAIPerHour; i++ {
		if !AllowPaidAI(ctx, rl, "user-1") {
			t.Fatalf("call %d for user-1 denied, want allowed", i+1)
		}
	}
	if AllowPaidAI(ctx, rl, "user-1") {
		t.Error("call 11 for user-1 allowed, want denied (over budget)")
	}

	// A different user has an independent budget.
	if !AllowPaidAI(ctx, rl, "user-2") {
		t.Error("user-2's first call denied, want allowed")
	}

	// Anonymous/system and nil-limiter always allowed (fail open).
	if !AllowPaidAI(ctx, rl, "") {
		t.Error("empty userID should always be allowed")
	}
	if !AllowPaidAI(ctx, nil, "user-1") {
		t.Error("nil limiter should fail open (allowed)")
	}
}
