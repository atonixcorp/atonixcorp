package analytics

import (
	"fmt"
	"strings"
)

// BAR CHART (██████░░░░░)
func Bar(value int, max int, width int) string {
	if max == 0 {
		return ""
	}
	filled := int(float64(value) / float64(max) * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// SPARKLINE (▁▂▃▄▅▆▇█)
func Spark(values []int) string {
	chars := []rune("▁▂▃▄▅▆▇█")
	if len(values) == 0 {
		return ""
	}

	max := 0
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	out := ""
	for _, v := range values {
		idx := int(float64(v) / float64(max) * 7)
		if idx > 7 {
			idx = 7
		}
		out += string(chars[idx])
	}

	return out
}

// TABLE RENDERER
func Table(headers []string, rows [][]string) string {
	out := ""

	// Render headers
	for _, h := range headers {
		out += fmt.Sprintf("%-20s", h)
	}
	out += "\n" + strings.Repeat("-", 20*len(headers)) + "\n"

	// Render rows
	for _, row := range rows {
		for _, col := range row {
			out += fmt.Sprintf("%-20s", col)
		}
		out += "\n"
	}

	return out
}
