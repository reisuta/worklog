// Package statusbar renders today's summary in SwiftBar's plugin format.
//
// SwiftBar reads stdout: the first line is the menu-bar text, "---" starts the
// dropdown, and "key | param=value" lines become clickable menu items.
package statusbar

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/format"
	"github.com/reisuta/worklog/internal/report"
	"github.com/reisuta/worklog/internal/sysstat"
)

// freshWindow is how recently a focus block must have ended to be treated as
// still in progress for the menu-bar line.
const freshWindow = 90 * time.Second

// Render returns the SwiftBar plugin output for the given summary. now is the
// current time (used to detect a focus block still in progress), binPath is the
// absolute path to the worklog binary for the action menu items, and sys (may
// be nil) carries optional system metrics to display.
func Render(s report.Summary, cfg config.Config, now time.Time, binPath string, sys *sysstat.Snapshot) string {
	var b strings.Builder

	// Menu-bar line. A live focus block takes priority over the plain total,
	// and memory usage is appended as the at-a-glance system headline.
	var head string
	if active, ok := activeFocus(s, now); ok {
		head = applyFocus(cfg.Statusbar.FocusFormat, active)
	} else {
		head = applyTotal(cfg.Statusbar.Format, s.TotalSec)
	}
	if sys != nil {
		head += systemHeadline(sys)
	}
	b.WriteString(head)
	b.WriteString("\n---\n")

	// Header: date and the three headline totals.
	fmt.Fprintf(&b, "%s\n", s.Range.Start.Format("2006-01-02 (Mon)"))
	fmt.Fprintf(&b, "合計 %s ｜ 集中 %s ｜ 離席 %s\n",
		format.Duration(s.TotalSec), format.Duration(s.FocusSec), format.Duration(s.IdleSec))
	b.WriteString("---\n")

	// Per-project breakdown, shown inline so no terminal is needed.
	if len(s.ByProject) == 0 {
		b.WriteString("まだ記録がありません\n")
	} else {
		b.WriteString("プロジェクト | size=11\n")
		for _, p := range s.TopProjects(8) {
			fmt.Fprintf(&b, "%s: %s\n", p.Name, format.Duration(p.Seconds))
		}
	}

	// Focus blocks, earliest first.
	if len(s.Focus) > 0 {
		b.WriteString("---\n")
		b.WriteString("集中ブロック | size=11\n")
		blocks := append([]report.Block(nil), s.Focus...)
		sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].Start.Before(blocks[j].Start) })
		for i, fb := range blocks {
			if i >= 6 {
				break
			}
			fmt.Fprintf(&b, "%s-%s  %s (%s)\n",
				fb.Start.Format("15:04"), fb.End.Format("15:04"),
				fb.Project, format.Duration(fb.Seconds()))
		}
	}

	// System metrics block (memory / CPU / battery).
	if sys != nil {
		writeSystem(&b, sys)
	}

	b.WriteString("---\n")
	fmt.Fprintf(&b, "ターミナルで詳細を見る | bash='%s' param1='today' terminal=true\n", binPath)
	fmt.Fprintf(&b, "停止 | bash='%s' param1='stop' terminal=false refresh=true\n", binPath)
	return b.String()
}

// systemHeadline renders the always-visible menu-bar metrics: memory (with a
// pressure-colored dot) and CPU.
func systemHeadline(sys *sysstat.Snapshot) string {
	var parts []string
	if m := sys.Memory; m != nil {
		parts = append(parts, fmt.Sprintf("%s %.0f%%", memoryDot(m.PressureLevel()), m.UsedPercent()))
	}
	if sys.CPU != nil {
		cpu := *sys.CPU
		flag := ""
		if cpu >= 90 {
			flag = " 🔴"
		} else if cpu >= 70 {
			flag = " 🟡"
		}
		parts = append(parts, fmt.Sprintf("⚙️ %.0f%%%s", cpu, flag))
	}
	if len(parts) == 0 {
		return ""
	}
	return "  " + strings.Join(parts, "  ")
}

// memoryDot maps a memory pressure level to a traffic-light emoji, matching
// Activity Monitor's green/yellow/red pressure indicator.
func memoryDot(level string) string {
	switch level {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "🧠" // pressure unknown
	}
}

// writeSystem appends the system metrics section to the dropdown.
func writeSystem(b *strings.Builder, sys *sysstat.Snapshot) {
	if sys.Memory == nil && sys.CPU == nil && sys.Battery == nil {
		return
	}
	b.WriteString("---\n")
	b.WriteString("システム | size=11\n")
	if m := sys.Memory; m != nil {
		fmt.Fprintf(b, "%s メモリ  %s / %s (%.0f%%, 圧迫%s)\n",
			memoryDot(m.PressureLevel()), format.Bytes(m.UsedBytes), format.Bytes(m.TotalBytes),
			m.UsedPercent(), pressureLabel(m.PressureLevel()))
	}
	if sys.CPU != nil {
		fmt.Fprintf(b, "⚙️ CPU  %.0f%%\n", *sys.CPU)
	}
	if bt := sys.Battery; bt != nil {
		if bt.State != "" {
			fmt.Fprintf(b, "🔋 バッテリー  %d%% (%s)\n", bt.Percent, bt.State)
		} else {
			fmt.Fprintf(b, "🔋 バッテリー  %d%%\n", bt.Percent)
		}
	}
}

// pressureLabel is the Japanese label for a memory pressure level.
func pressureLabel(level string) string {
	switch level {
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return "?"
	}
}

// activeFocus returns the most recent focus block if it ended recently enough
// (within freshWindow of now) to be considered still in progress.
func activeFocus(s report.Summary, now time.Time) (report.Block, bool) {
	if len(s.Focus) == 0 {
		return report.Block{}, false
	}
	last := s.Focus[0]
	for _, b := range s.Focus {
		if b.End.After(last.End) {
			last = b
		}
	}
	if now.Sub(last.End) <= freshWindow {
		return last, true
	}
	return report.Block{}, false
}

func applyTotal(tmpl string, totalSec int) string {
	if tmpl == "" {
		tmpl = "⏱ {total}"
	}
	return strings.ReplaceAll(tmpl, "{total}", format.Duration(totalSec))
}

func applyFocus(tmpl string, b report.Block) string {
	if tmpl == "" {
		tmpl = "🔥 {duration} ({project})"
	}
	out := strings.ReplaceAll(tmpl, "{duration}", format.Duration(b.Seconds()))
	out = strings.ReplaceAll(out, "{project}", b.Project)
	return out
}
