// Package report builds summaries and focus-block analyses from stored events
// and renders them as text or Markdown.
package report

import "time"

// Range is a half-open time window [Start, End) used by every report.
type Range struct {
	Start time.Time
	End   time.Time
}

// Unix returns the range bounds as Unix seconds.
func (r Range) Unix() (from, to int64) {
	return r.Start.Unix(), r.End.Unix()
}

// Day returns the local-day window containing t.
func Day(t time.Time) Range {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return Range{Start: start, End: start.AddDate(0, 0, 1)}
}

// Week returns the Monday-to-Monday window containing t.
func Week(t time.Time) Range {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	// Go's Weekday() is Sunday=0; shift so Monday is the first day.
	offset := (int(start.Weekday()) + 6) % 7
	start = start.AddDate(0, 0, -offset)
	return Range{Start: start, End: start.AddDate(0, 0, 7)}
}

// Month returns the calendar-month window containing t.
func Month(t time.Time) Range {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return Range{Start: start, End: start.AddDate(0, 1, 0)}
}

// Custom returns the window [from, to) spanning whole local days.
func Custom(from, to time.Time) Range {
	s := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	e := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location()).AddDate(0, 0, 1)
	return Range{Start: s, End: e}
}
