package statusbar

import (
	"strings"
	"testing"
	"time"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/report"
	"github.com/reisuta/worklog/internal/store"
	"github.com/reisuta/worklog/internal/sysstat"
)

func day(h, m int) time.Time {
	return time.Date(2026, 5, 29, h, m, 0, 0, time.Local)
}

func TestRenderShowsDetailAndActions(t *testing.T) {
	s := report.Summary{
		Range:    report.Range{Start: day(0, 0), End: day(0, 0).AddDate(0, 0, 1)},
		TotalSec: 4*3600 + 23*60,
		IdleSec:  65 * 60,
		FocusSec: 2*3600 + 15*60,
		ByProject: []store.Bucket{
			{Name: "blog", Seconds: 2*3600 + 30*60},
			{Name: "sns", Seconds: 25 * 60},
		},
		Focus: []report.Block{
			{Start: day(10, 15), End: day(11, 23), App: "Cursor", Project: "blog"},
		},
	}
	out := Render(s, config.Default(), day(18, 0), "/usr/local/bin/worklog", nil)

	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "⏱") {
		t.Errorf("menu-bar line should show the total, got %q", firstLine)
	}
	for _, want := range []string{
		"blog", "集中 2h 15m", "集中ブロック", "10:15-11:23",
		"param1='today'", "param1='stop'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "param1='ui'") {
		t.Error("status bar should no longer link to the unimplemented ui command")
	}
}

func TestRenderActiveFocusInMenuBar(t *testing.T) {
	s := report.Summary{
		Range:    report.Range{Start: day(0, 0), End: day(0, 0).AddDate(0, 0, 1)},
		TotalSec: 3600,
		Focus: []report.Block{
			{Start: day(11, 0), End: day(11, 23), App: "Cursor", Project: "blog"},
		},
	}
	// now is 30s after the block ended -> still "in progress".
	out := Render(s, config.Default(), day(11, 23).Add(30*time.Second), "worklog", nil)
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "🔥") || !strings.Contains(firstLine, "blog") {
		t.Errorf("expected active focus line with 🔥 and project, got %q", firstLine)
	}
}

func TestRenderSystemBlock(t *testing.T) {
	s := report.Summary{
		Range:    report.Range{Start: day(0, 0), End: day(0, 0).AddDate(0, 0, 1)},
		TotalSec: 3600,
	}
	cpu := 18.0
	sys := &sysstat.Snapshot{
		Memory:  &sysstat.MemStat{TotalBytes: 16 << 30, UsedBytes: 8 << 30, FreePercent: 60}, // 50%, low pressure -> 🟢
		CPU:     &cpu,
		Battery: &sysstat.BatteryStat{Present: true, Percent: 80, State: "charging"},
	}
	out := Render(s, config.Default(), day(12, 0), "worklog", sys)

	firstLine := strings.SplitN(out, "\n", 2)[0]
	for _, want := range []string{"🟢 50%", "⚙️ 18%"} { // memory + CPU always in the menu bar
		if !strings.Contains(firstLine, want) {
			t.Errorf("menu-bar line missing %q, got %q", want, firstLine)
		}
	}
	for _, want := range []string{"システム", "メモリ", "圧迫低", "⚙️ CPU  18%", "🔋 バッテリー  80% (charging)"} {
		if !strings.Contains(out, want) {
			t.Errorf("system block missing %q\n---\n%s", want, out)
		}
	}
}

func TestMemoryDotByPressure(t *testing.T) {
	tests := []struct {
		free int
		want string
	}{
		{60, "🟢"}, {20, "🟡"}, {5, "🔴"}, {-1, "🧠"},
	}
	for _, tt := range tests {
		m := sysstat.MemStat{TotalBytes: 100, UsedBytes: 50, FreePercent: tt.free}
		if got := memoryDot(m.PressureLevel()); got != tt.want {
			t.Errorf("free=%d -> %q, want %q", tt.free, got, tt.want)
		}
	}
}
