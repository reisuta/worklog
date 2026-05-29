package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reisuta/worklog/internal/daemon"
	"github.com/reisuta/worklog/internal/format"
	"github.com/reisuta/worklog/internal/report"
	"github.com/reisuta/worklog/internal/statusbar"
	"github.com/reisuta/worklog/internal/sysstat"
	"github.com/reisuta/worklog/internal/tracker"
)

func cmdStart(e env) error {
	pid, err := daemon.Start(e.cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "worklog started (pid %d)\n", pid)
	return nil
}

func cmdStop(e env) error {
	if err := daemon.Stop(e.cfg); err != nil {
		return err
	}
	fmt.Fprintln(e.out, "worklog stopped")
	return nil
}

func cmdStatus(e env) error {
	st, err := daemon.Query(e.cfg)
	if err != nil {
		return err
	}
	if st.Running {
		fmt.Fprintf(e.out, "running (pid %d)\n", st.PID)
	} else {
		fmt.Fprintln(e.out, "not running")
	}
	return nil
}

// cmdDaemon runs the tracker in the foreground until terminated. It is used
// both by `worklog start` (via fork) and by launchd.
func cmdDaemon(e env) error {
	s, err := e.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	// Record our PID so `status`/`stop` work even under launchd.
	pidPath := daemon.PIDPath(e.cfg)
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644)
	defer os.Remove(pidPath)

	logger := log.New(os.Stderr, "", log.LstdFlags)
	t := tracker.New(s, e.rules, e.cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = t.Run(ctx)
	if err == context.Canceled {
		return nil
	}
	return err
}

func cmdToday(e env) error {
	return printSummary(e, report.Day(time.Now()), "今日 ("+time.Now().Format("2006-01-02")+")")
}

func cmdYesterday(e env) error {
	y := time.Now().AddDate(0, 0, -1)
	return printSummary(e, report.Day(y), "昨日 ("+y.Format("2006-01-02")+")")
}

func cmdWeek(e env) error {
	r := report.Week(time.Now())
	title := fmt.Sprintf("今週 (%s 〜 %s)", r.Start.Format("01-02"), r.End.AddDate(0, 0, -1).Format("01-02"))
	return printSummary(e, r, title)
}

func cmdMonth(e env) error {
	r := report.Month(time.Now())
	return printSummary(e, r, "今月 ("+r.Start.Format("2006-01")+")")
}

func printSummary(e env, r report.Range, title string) error {
	s, err := e.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	sum, err := report.Build(s, r, e.cfg.Tracker.PollIntervalSeconds, e.cfg.Tracker.FocusBlockMinMinutes)
	if err != nil {
		return err
	}
	fmt.Fprint(e.out, sum.Text(title))
	return nil
}

func cmdStatusbar(e env) error {
	s, err := e.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	now := time.Now()
	sum, err := report.Build(s, report.Day(now), e.cfg.Tracker.PollIntervalSeconds, e.cfg.Tracker.FocusBlockMinMinutes)
	if err != nil {
		return err
	}
	var sys *sysstat.Snapshot
	if e.cfg.Statusbar.ShowSystem {
		snap := sysstat.Collect()
		sys = &snap
	}
	exe, _ := os.Executable()
	fmt.Fprint(e.out, statusbar.Render(sum, e.cfg, now, exe, sys))
	return nil
}

// cmdStats prints current system resource usage to the terminal.
func cmdStats(e env) error {
	snap := sysstat.Collect()
	if snap.Memory != nil {
		m := *snap.Memory
		pressure := map[string]string{"high": "高", "medium": "中", "low": "低", "unknown": "?"}[m.PressureLevel()]
		fmt.Fprintf(e.out, "メモリ:      %s / %s (%.0f%%, 圧迫%s)\n",
			format.Bytes(m.UsedBytes), format.Bytes(m.TotalBytes), m.UsedPercent(), pressure)
	}
	if snap.CPU != nil {
		fmt.Fprintf(e.out, "CPU:         %.0f%%\n", *snap.CPU)
	}
	if snap.Battery != nil {
		if snap.Battery.State != "" {
			fmt.Fprintf(e.out, "バッテリー:  %d%% (%s)\n", snap.Battery.Percent, snap.Battery.State)
		} else {
			fmt.Fprintf(e.out, "バッテリー:  %d%%\n", snap.Battery.Percent)
		}
	}
	if snap.Memory == nil && snap.CPU == nil && snap.Battery == nil {
		fmt.Fprintln(e.out, "システム情報を取得できませんでした。")
	}
	return nil
}

func cmdUI(e env) error {
	fmt.Fprintln(e.out, "TUI ダッシュボード (worklog ui) は今後のリリースで実装予定です。")
	fmt.Fprintln(e.out, "現在は `worklog today` / `worklog week` をご利用ください。")
	return nil
}

func cmdInstallLaunchd(e env) error {
	path, err := daemon.InstallLaunchd(e.cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "launchd に登録しました: %s\n", path)
	fmt.Fprintln(e.out, "初回はアクセシビリティ権限の許可が必要です (システム設定 > プライバシーとセキュリティ)。")
	return nil
}

func cmdUninstallLaunchd(e env) error {
	path, err := daemon.UninstallLaunchd()
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "launchd の登録を解除しました: %s\n", path)
	return nil
}
