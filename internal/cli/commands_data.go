package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/export"
	"github.com/reisuta/worklog/internal/report"
)

const dateLayout = "2006-01-02"

func cmdReport(e env, args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(e.errw)
	from := fs.String("from", "", "開始日 (YYYY-MM-DD)")
	to := fs.String("to", "", "終了日 (YYYY-MM-DD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return errors.New("report は --from と --to の両方が必要です")
	}
	fromT, err := time.ParseInLocation(dateLayout, *from, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --from: %w", err)
	}
	toT, err := time.ParseInLocation(dateLayout, *to, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --to: %w", err)
	}
	if toT.Before(fromT) {
		return errors.New("--to は --from 以降の日付にしてください")
	}
	r := report.Custom(fromT, toT)
	title := fmt.Sprintf("レポート (%s 〜 %s)", *from, *to)
	return printSummary(e, r, title)
}

func cmdRule(e env, args []string) error {
	sub := "list"
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "list":
		return ruleList(e)
	case "edit":
		return openInEditor(e, config.RulesPath(), sampleRules)
	default:
		return fmt.Errorf("unknown rule subcommand %q (list|edit)", sub)
	}
}

func ruleList(e env) error {
	if len(e.rules.Rules) == 0 {
		fmt.Fprintln(e.out, "ルールがありません。`worklog rule edit` で追加できます。")
	}
	for i, r := range e.rules.Rules {
		app := r.MatchApp
		if app == "" {
			app = "*"
		}
		title := r.MatchTitle
		if title == "" {
			title = "*"
		}
		fmt.Fprintf(e.out, "%d. app=%s title=%s → project=%s category=%s\n",
			i+1, app, title, r.Project, r.Category)
	}
	fmt.Fprintf(e.out, "default → project=%s category=%s\n", e.rules.Default.Project, e.rules.Default.Category)
	return nil
}

func cmdExport(e env, args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(e.errw)
	format := fs.String("format", "csv", "出力形式 (csv|json)")
	month := fs.String("month", "", "対象月 (YYYY-MM)")
	from := fs.String("from", "", "開始日 (YYYY-MM-DD)")
	to := fs.String("to", "", "終了日 (YYYY-MM-DD)")
	week := fs.Bool("week", false, "今週を対象にする")
	if err := fs.Parse(args); err != nil {
		return err
	}

	r, err := resolveRange(*month, *from, *to, *week)
	if err != nil {
		return err
	}

	s, err := e.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	fromTS, toTS := r.Unix()
	events, err := s.Events(fromTS, toTS)
	if err != nil {
		return err
	}

	switch strings.ToLower(*format) {
	case "csv":
		return export.CSV(e.out, events)
	case "json":
		return export.JSON(e.out, events)
	default:
		return fmt.Errorf("unknown format %q (csv|json)", *format)
	}
}

// resolveRange picks a time range from the export/report flags.
func resolveRange(month, from, to string, week bool) (report.Range, error) {
	switch {
	case month != "":
		t, err := time.ParseInLocation("2006-01", month, time.Local)
		if err != nil {
			return report.Range{}, fmt.Errorf("invalid --month: %w", err)
		}
		return report.Month(t), nil
	case week:
		return report.Week(time.Now()), nil
	case from != "" && to != "":
		f, err := time.ParseInLocation(dateLayout, from, time.Local)
		if err != nil {
			return report.Range{}, fmt.Errorf("invalid --from: %w", err)
		}
		t, err := time.ParseInLocation(dateLayout, to, time.Local)
		if err != nil {
			return report.Range{}, fmt.Errorf("invalid --to: %w", err)
		}
		return report.Custom(f, t), nil
	default:
		// Default to the current month.
		return report.Month(time.Now()), nil
	}
}

func cmdWipe(e env, args []string) error {
	fs := flag.NewFlagSet("wipe", flag.ContinueOnError)
	fs.SetOutput(e.errw)
	before := fs.String("before", "", "この日付より前を削除 (YYYY-MM-DD)")
	all := fs.Bool("all", false, "すべて削除")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*all && *before == "" {
		return errors.New("wipe は --before または --all を指定してください")
	}

	s, err := e.openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if *all {
		n, err := s.WipeAll()
		if err != nil {
			return err
		}
		fmt.Fprintf(e.out, "%d 件のイベントを削除しました (全件)\n", n)
		return nil
	}

	t, err := time.ParseInLocation(dateLayout, *before, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --before: %w", err)
	}
	n, err := s.Wipe(t.Unix())
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "%d 件のイベントを削除しました (%s より前)\n", n, *before)
	return nil
}

func cmdConfig(e env) error {
	return openInEditor(e, config.Path(), sampleConfig)
}

// openInEditor opens path in $EDITOR, creating it with sample content first if
// it does not yet exist.
func openInEditor(e env, path, sample string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(e.out, "新規作成しました: %s\n", path)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
