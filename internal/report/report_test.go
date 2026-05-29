package report

import (
	"testing"
	"time"

	"github.com/reisuta/worklog/internal/store"
)

func TestWeekStartsMonday(t *testing.T) {
	// 2026-05-29 is a Friday.
	fri := time.Date(2026, 5, 29, 14, 0, 0, 0, time.Local)
	r := Week(fri)
	if r.Start.Weekday() != time.Monday {
		t.Errorf("week start = %s, want Monday", r.Start.Weekday())
	}
	if r.Start.Day() != 25 { // Monday 2026-05-25
		t.Errorf("week start day = %d, want 25", r.Start.Day())
	}
	if d := r.End.Sub(r.Start); d != 7*24*time.Hour {
		t.Errorf("week length = %s, want 168h", d)
	}
}

func TestDayRange(t *testing.T) {
	noon := time.Date(2026, 5, 29, 12, 30, 0, 0, time.Local)
	r := Day(noon)
	if r.Start.Hour() != 0 || r.Start.Day() != 29 {
		t.Errorf("day start = %v", r.Start)
	}
	if r.End.Day() != 30 {
		t.Errorf("day end = %v, want next day", r.End)
	}
}

func TestMonthRange(t *testing.T) {
	r := Month(time.Date(2026, 5, 29, 0, 0, 0, 0, time.Local))
	if r.Start.Day() != 1 || r.Start.Month() != time.May {
		t.Errorf("month start = %v", r.Start)
	}
	if r.End.Month() != time.June || r.End.Day() != 1 {
		t.Errorf("month end = %v", r.End)
	}
}

func TestFocusBlocks(t *testing.T) {
	const interval = 5
	base := time.Date(2026, 5, 29, 10, 0, 0, 0, time.Local).Unix()
	mk := func(offsetSec int, proj string, idle bool) store.Event {
		return store.Event{TS: base + int64(offsetSec), App: "Cursor", Project: proj, IsIdle: idle}
	}

	// 0..600s on blog (601s span >= 600s threshold for 10 min) then a gap,
	// then a short sns burst that should not qualify.
	var events []store.Event
	for s := 0; s <= 600; s += interval {
		events = append(events, mk(s, "blog", false))
	}
	// big gap (idle) then short sns
	events = append(events, mk(1000, "sns", false))
	events = append(events, mk(1005, "sns", false))

	blocks := FocusBlocks(events, interval, 10) // 10 min minimum
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1: %+v", len(blocks), blocks)
	}
	if blocks[0].Project != "blog" {
		t.Errorf("block project = %q, want blog", blocks[0].Project)
	}
	if blocks[0].Seconds() < 600 {
		t.Errorf("block seconds = %d, want >= 600", blocks[0].Seconds())
	}
}

func TestFocusBlocksBrokenByIdle(t *testing.T) {
	const interval = 5
	base := int64(0)
	var events []store.Event
	for s := 0; s < 300; s += interval {
		events = append(events, store.Event{TS: base + int64(s), Project: "blog"})
	}
	// idle event in the middle then resume
	events = append(events, store.Event{TS: 300, Project: "blog", IsIdle: true})
	for s := 305; s < 600; s += interval {
		events = append(events, store.Event{TS: int64(s), Project: "blog"})
	}
	// Neither half reaches 10 minutes, so no qualifying block.
	if blocks := FocusBlocks(events, interval, 10); len(blocks) != 0 {
		t.Errorf("expected idle to split blocks below threshold, got %+v", blocks)
	}
}
