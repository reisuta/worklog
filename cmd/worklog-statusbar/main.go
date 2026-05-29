// Command worklog-statusbar prints today's summary in SwiftBar plugin format.
//
// Symlink it into SwiftBar's plugin directory with a refresh interval, e.g.:
//
//	ln -s "$(which worklog-statusbar)" ~/SwiftBar/worklog.5s.bin
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/report"
	"github.com/reisuta/worklog/internal/statusbar"
	"github.com/reisuta/worklog/internal/store"
)

func main() {
	if err := run(); err != nil {
		// SwiftBar shows stdout in the menu bar, so surface errors there too.
		fmt.Println("⏱ —")
		fmt.Println("---")
		fmt.Fprintln(os.Stderr, "worklog-statusbar:", err)
		fmt.Println("error:", err)
		os.Exit(0)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	s, err := store.Open(cfg.ResolvedDBPath())
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	sum, err := report.Build(s, report.Day(now), cfg.Tracker.PollIntervalSeconds, cfg.Tracker.FocusBlockMinMinutes)
	if err != nil {
		return err
	}

	// Prefer the main worklog binary for menu actions if it's alongside us.
	bin := "worklog"
	if exe, err := os.Executable(); err == nil {
		if cand := siblingWorklog(exe); cand != "" {
			bin = cand
		}
	}
	fmt.Print(statusbar.Render(sum, cfg, now, bin))
	return nil
}

// siblingWorklog returns the path to a `worklog` binary next to exe, if present.
func siblingWorklog(exe string) string {
	dir := dirOf(exe)
	cand := dir + "/worklog"
	if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
		return cand
	}
	return ""
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
