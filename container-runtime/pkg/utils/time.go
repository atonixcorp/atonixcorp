package utils

import "time"

// NowUTC returns the current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatTimestamp formats a time into a standard string.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
