// Package format holds small, dependency-free helpers for rendering
// durations and bar charts that are shared across the CLI, reports and
// the SwiftBar status line.
package format

import (
	"fmt"
	"strings"
)

// Duration renders a number of seconds as a compact human string such as
// "4h 23m", "23m" or "45s". It is intentionally terse so it fits in a menu
// bar.
func Duration(sec int) string {
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// Bytes renders a byte count as gigabytes, e.g. "12.4 GB". Memory figures are
// always GB-scale so a fixed unit keeps the status bar from jumping around.
func Bytes(b uint64) string {
	gb := float64(b) / (1 << 30)
	return fmt.Sprintf("%.1f GB", gb)
}

// Bar renders a proportional bar of the given width using block characters.
// value/total determines how much of the bar is filled.
func Bar(value, total, width int) string {
	if width <= 0 {
		return ""
	}
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	filled := value * width / total
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
