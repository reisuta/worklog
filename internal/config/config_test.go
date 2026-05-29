package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissingReturnsDefault(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("LoadFrom missing: %v", err)
	}
	def := Default()
	if cfg.Tracker.PollIntervalSeconds != def.Tracker.PollIntervalSeconds {
		t.Errorf("missing file should yield defaults, got %d", cfg.Tracker.PollIntervalSeconds)
	}
}

func TestLoadFromOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[tracker]
poll_interval_seconds = 10

[privacy]
record_window_title = false
exclude_apps = ["KeePassXC"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Tracker.PollIntervalSeconds != 10 {
		t.Errorf("poll interval = %d, want 10", cfg.Tracker.PollIntervalSeconds)
	}
	// Unset fields keep their defaults.
	if cfg.Tracker.IdleThresholdSeconds != 300 {
		t.Errorf("idle threshold = %d, want default 300", cfg.Tracker.IdleThresholdSeconds)
	}
	if cfg.Privacy.RecordWindowTitle {
		t.Error("record_window_title should be false")
	}
}

func TestNormalizeRepairsZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[tracker]\npoll_interval_seconds = 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tracker.PollIntervalSeconds != 5 {
		t.Errorf("zero poll interval should be repaired to 5, got %d", cfg.Tracker.PollIntervalSeconds)
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := Default()
	cfg.Privacy.ExcludeApps = []string{"1Password", "KeePassXC"}
	if !cfg.IsExcluded("1password") {
		t.Error("IsExcluded should be case-insensitive")
	}
	if cfg.IsExcluded("Cursor") {
		t.Error("Cursor should not be excluded")
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := ExpandHome("~/foo"); got != filepath.Join(home, "foo") {
		t.Errorf("ExpandHome(~/foo) = %q", got)
	}
	if got := ExpandHome("/abs/path"); got != "/abs/path" {
		t.Errorf("ExpandHome should leave absolute paths, got %q", got)
	}
}
