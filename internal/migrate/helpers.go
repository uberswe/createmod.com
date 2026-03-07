package migrate

import (
	"encoding/json"
	"strings"
	"time"
)

// parsePBTimestamp parses PocketBase's timestamp format into a *time.Time.
// Returns nil for empty strings.
func parsePBTimestamp(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// PocketBase format: "2024-01-15 12:30:45.123Z"
	formats := []string{
		"2006-01-02 15:04:05.999Z",
		"2006-01-02 15:04:05.999Z07:00",
		"2006-01-02 15:04:05.000Z",
		"2006-01-02 15:04:05Z",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return &t
		}
	}
	return nil
}

// parsePBTimestampRequired parses a PocketBase timestamp, returning time.Now() as fallback.
func parsePBTimestampRequired(s string) time.Time {
	t := parsePBTimestamp(s)
	if t == nil {
		return time.Now()
	}
	return *t
}

// parseJSONStringArray parses a PocketBase JSON array of strings like '["id1","id2"]'.
// Returns nil for empty/invalid input.
func parseJSONStringArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	return result
}

// sqliteBool converts SQLite 0/1 to Go bool.
func sqliteBool(v interface{}) bool {
	switch val := v.(type) {
	case int64:
		return val != 0
	case bool:
		return val
	case string:
		return val == "1" || strings.EqualFold(val, "true")
	default:
		return false
	}
}

// nullStr returns nil if string is empty, otherwise a pointer to it.
func nullStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// nilIfZero returns nil if v is 0, otherwise a pointer to v.
func nilIfZero(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}
