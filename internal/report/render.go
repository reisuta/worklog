package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/reisuta/worklog/internal/format"
	"github.com/reisuta/worklog/internal/store"
)

// Text renders a summary as a plain-text report for the terminal. title is the
// heading, e.g. "今日 (2026-05-29)".
func (s Summary) Text(title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", title)
	fmt.Fprintf(&b, "合計 %s (集中 %s、離席 %s)\n\n",
		format.Duration(s.TotalSec), format.Duration(s.FocusSec), format.Duration(s.IdleSec))

	if len(s.ByProject) == 0 {
		b.WriteString("記録がありません。\n")
		return b.String()
	}

	b.WriteString("プロジェクト\n")
	writeBuckets(&b, s.ByProject, s.TotalSec)

	if len(s.Focus) > 0 {
		b.WriteString("\n集中ブロック\n")
		blocks := append([]Block(nil), s.Focus...)
		sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].Start.Before(blocks[j].Start) })
		for _, fb := range blocks {
			fmt.Fprintf(&b, "  %s-%s (%s) %s %s\n",
				fb.Start.Format("15:04"), fb.End.Format("15:04"),
				format.Duration(fb.Seconds()), fb.Project, fb.App)
		}
	}
	return b.String()
}

func writeBuckets(b *strings.Builder, buckets []store.Bucket, total int) {
	maxName := 0
	for _, k := range buckets {
		if len(k.Name) > maxName {
			maxName = len(k.Name)
		}
	}
	for _, k := range buckets {
		fmt.Fprintf(b, "  %-*s  %8s  %s\n",
			maxName, k.Name, format.Duration(k.Seconds), format.Bar(k.Seconds, total, 12))
	}
}

// Markdown renders a summary as a Markdown document for archival reports.
func (s Summary) Markdown(title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "- 合計: **%s**\n", format.Duration(s.TotalSec))
	fmt.Fprintf(&b, "- 集中: %s\n", format.Duration(s.FocusSec))
	fmt.Fprintf(&b, "- 離席: %s\n\n", format.Duration(s.IdleSec))

	if len(s.ByProject) > 0 {
		b.WriteString("## プロジェクト\n\n| プロジェクト | 時間 |\n|---|---|\n")
		for _, k := range s.ByProject {
			fmt.Fprintf(&b, "| %s | %s |\n", k.Name, format.Duration(k.Seconds))
		}
		b.WriteString("\n")
	}

	if len(s.ByApp) > 0 {
		b.WriteString("## アプリ\n\n| アプリ | 時間 |\n|---|---|\n")
		for _, k := range s.ByApp {
			fmt.Fprintf(&b, "| %s | %s |\n", k.Name, format.Duration(k.Seconds))
		}
		b.WriteString("\n")
	}

	if len(s.Focus) > 0 {
		b.WriteString("## 集中ブロック\n\n| 時間帯 | 長さ | プロジェクト | アプリ |\n|---|---|---|---|\n")
		blocks := append([]Block(nil), s.Focus...)
		sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].Start.Before(blocks[j].Start) })
		for _, fb := range blocks {
			fmt.Fprintf(&b, "| %s-%s | %s | %s | %s |\n",
				fb.Start.Format("15:04"), fb.End.Format("15:04"),
				format.Duration(fb.Seconds()), fb.Project, fb.App)
		}
	}
	return b.String()
}
