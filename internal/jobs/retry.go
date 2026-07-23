package jobs

import "time"

// slowRetryAt implements a high-backoff retry schedule for workers whose
// failures usually mean an external dependency (OpenAI, mail) is unavailable:
// retrying quickly just burns attempts against the same outage. Delays double
// from 30 minutes up to a 24-hour ceiling: 30m, 1h, 2h, 4h, 8h, 16h, 24h,
// 24h, ... attempt is the attempt that just failed (1-based).
func slowRetryAt(attempt int, now time.Time) time.Time {
	delay := 30 * time.Minute
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= 24*time.Hour {
			delay = 24 * time.Hour
			break
		}
	}
	return now.Add(delay)
}
