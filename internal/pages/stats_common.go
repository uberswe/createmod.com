package pages

import "time"

// HourlyTrackingCutoff is the date when hourly view tracking (type='5') was deployed.
// VD ratio calculations should only use data after this date.
var HourlyTrackingCutoff = time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
