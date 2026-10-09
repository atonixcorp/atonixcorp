package utils

import "strings"

// Trim removes whitespace from both ends of a string.
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// IsEmpty checks if a string is empty after trimming.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// Join joins multiple strings with a separator.
func Join(sep string, parts ...string) string {
	return strings.Join(parts, sep)
}
