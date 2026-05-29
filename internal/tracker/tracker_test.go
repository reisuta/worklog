package tracker

import (
	"testing"

	"github.com/reisuta/worklog/internal/config"
	"github.com/reisuta/worklog/internal/rules"
)

func testTracker(t *testing.T, cfg config.Config, rs rules.Set) *Tracker {
	t.Helper()
	return &Tracker{rules: rs, cfg: cfg}
}

func TestEventIdleThreshold(t *testing.T) {
	cfg := config.Default()
	cfg.Tracker.IdleThresholdSeconds = 300
	tr := testTracker(t, cfg, rules.DefaultSet())

	if e := tr.event(Sample{App: "Cursor", Title: "x", IdleSec: 299}, 100); e.IsIdle {
		t.Error("299s should not be idle")
	}
	if e := tr.event(Sample{App: "Cursor", Title: "x", IdleSec: 300}, 100); !e.IsIdle {
		t.Error("300s should be idle")
	}
}

func TestEventPrivacyTitleStripped(t *testing.T) {
	cfg := config.Default()
	cfg.Privacy.RecordWindowTitle = false
	tr := testTracker(t, cfg, rules.DefaultSet())

	if e := tr.event(Sample{App: "Cursor", Title: "secret", IdleSec: 0}, 1); e.Title != "" {
		t.Errorf("title should be stripped when recording disabled, got %q", e.Title)
	}
}

func TestEventExcludedApp(t *testing.T) {
	cfg := config.Default()
	cfg.Privacy.RecordWindowTitle = true
	cfg.Privacy.ExcludeApps = []string{"1Password"}
	tr := testTracker(t, cfg, rules.DefaultSet())

	e := tr.event(Sample{App: "1Password", Title: "master pw", IdleSec: 0}, 1)
	if e.Title != "" {
		t.Errorf("excluded app title should be blank, got %q", e.Title)
	}
	if e.App != "1Password" {
		t.Errorf("app name should still be recorded, got %q", e.App)
	}
}

func TestEventRuleApplied(t *testing.T) {
	rs, err := rules.Parse([]byte(`
[[rules]]
match_app = "Slack"
project = "communication"
category = "work"
`))
	if err != nil {
		t.Fatal(err)
	}
	tr := testTracker(t, config.Default(), rs)
	e := tr.event(Sample{App: "Slack", Title: "general", IdleSec: 0}, 1)
	if e.Project != "communication" || e.Category != "work" {
		t.Errorf("rule not applied: got project=%q category=%q", e.Project, e.Category)
	}
}
