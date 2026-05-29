// Package cli implements the worklog command-line interface.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/rules"
	"github.com/reisuta/worklog/internal/store"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// env bundles the loaded configuration and rules shared by commands.
type env struct {
	cfg   config.Config
	rules rules.Set
	out   io.Writer
	errw  io.Writer
}

func loadEnv() (env, error) {
	cfg, err := config.Load()
	if err != nil {
		return env{}, err
	}
	rs, err := rules.Load(config.RulesPath())
	if err != nil {
		return env{}, err
	}
	return env{cfg: cfg, rules: rs, out: os.Stdout, errw: os.Stderr}, nil
}

// openStore opens the database described by the loaded config.
func (e env) openStore() (*store.Store, error) {
	return store.Open(e.cfg.ResolvedDBPath())
}

// Main is the entry point. It returns a process exit code.
func Main(args []string) int {
	if len(args) == 0 {
		usage(os.Stdout)
		return 0
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage(os.Stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Println("worklog", Version)
		return 0
	}

	e, err := loadEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "worklog:", err)
		return 1
	}

	var cmdErr error
	switch cmd {
	case "start":
		cmdErr = cmdStart(e)
	case "stop":
		cmdErr = cmdStop(e)
	case "status":
		cmdErr = cmdStatus(e)
	case "daemon":
		cmdErr = cmdDaemon(e)
	case "today":
		cmdErr = cmdToday(e)
	case "yesterday":
		cmdErr = cmdYesterday(e)
	case "week":
		cmdErr = cmdWeek(e)
	case "month":
		cmdErr = cmdMonth(e)
	case "report":
		cmdErr = cmdReport(e, rest)
	case "statusbar":
		cmdErr = cmdStatusbar(e)
	case "stats":
		cmdErr = cmdStats(e)
	case "ui":
		cmdErr = cmdUI(e)
	case "rule":
		cmdErr = cmdRule(e, rest)
	case "export":
		cmdErr = cmdExport(e, rest)
	case "wipe":
		cmdErr = cmdWipe(e, rest)
	case "install-launchd":
		cmdErr = cmdInstallLaunchd(e)
	case "uninstall-launchd":
		cmdErr = cmdUninstallLaunchd(e)
	case "config":
		cmdErr = cmdConfig(e)
	default:
		fmt.Fprintf(os.Stderr, "worklog: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}

	if cmdErr != nil {
		fmt.Fprintln(os.Stderr, "worklog:", cmdErr)
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, `worklog — Mac で「何に時間を使ったか」を自動記録する CLI / デーモン

USAGE:
  worklog <command> [flags]

DAEMON:
  start                 デーモンをバックグラウンド起動
  stop                  デーモンを停止
  status                動作状態を表示
  daemon                フォアグラウンド実行 (launchd / 内部用)
  install-launchd       ログイン時に自動起動するよう登録
  uninstall-launchd     自動起動の登録を解除

REPORTS:
  today                 当日サマリ
  yesterday             前日サマリ
  week                  今週サマリ (月曜始まり)
  month                 今月サマリ
  report --from D --to D  期間指定レポート (D = YYYY-MM-DD)
  statusbar             SwiftBar 形式で当日サマリを出力
  stats                 メモリ / CPU / バッテリーの現在値を表示
  ui                    TUI ダッシュボード (未実装)

DATA:
  export --format csv|json [--month YYYY-MM | --week | --from D --to D]
  wipe --before YYYY-MM-DD | --all

CONFIG:
  rule list             ルール一覧
  rule edit             $EDITOR で rules.toml を開く
  config                $EDITOR で config.toml を開く

  version               バージョン表示
  help                  このヘルプ

データはすべてローカル (~/.local/share/worklog) に保存されます。
`)
}
