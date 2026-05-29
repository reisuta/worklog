// Package config loads and represents the user configuration for worklog.
//
// The configuration lives at ~/.config/worklog/config.toml. Missing files and
// missing fields fall back to sane defaults so worklog runs out of the box.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Config is the top-level configuration.
type Config struct {
	Tracker   Tracker   `toml:"tracker"`
	Storage   Storage   `toml:"storage"`
	Reports   Reports   `toml:"reports"`
	Statusbar Statusbar `toml:"statusbar"`
	Privacy   Privacy   `toml:"privacy"`
}

// Tracker controls the monitoring loop.
type Tracker struct {
	PollIntervalSeconds  int `toml:"poll_interval_seconds"`
	IdleThresholdSeconds int `toml:"idle_threshold_seconds"`
	FocusBlockMinMinutes int `toml:"focus_block_min_minutes"`
}

// Storage controls where data is persisted.
type Storage struct {
	DBPath string `toml:"db_path"`
}

// Reports controls report generation.
type Reports struct {
	AutoDailyMarkdown bool   `toml:"auto_daily_markdown"`
	ReportDir         string `toml:"report_dir"`
}

// Statusbar controls SwiftBar rendering.
type Statusbar struct {
	Format      string `toml:"format"`
	FocusFormat string `toml:"focus_format"`
	ShowSystem  bool   `toml:"show_system"` // show memory/CPU/battery block
}

// Privacy controls what is recorded.
type Privacy struct {
	RecordWindowTitle bool     `toml:"record_window_title"`
	ExcludeApps       []string `toml:"exclude_apps"`
}

// Default returns a Config populated with the documented defaults.
func Default() Config {
	return Config{
		Tracker: Tracker{
			PollIntervalSeconds:  5,
			IdleThresholdSeconds: 300,
			FocusBlockMinMinutes: 15,
		},
		Storage: Storage{
			DBPath: "~/.local/share/worklog/worklog.db",
		},
		Reports: Reports{
			AutoDailyMarkdown: false,
			ReportDir:         "~/Documents/worklog",
		},
		Statusbar: Statusbar{
			Format:      "⏱ {total}",
			FocusFormat: "🔥 {duration} ({project})",
			ShowSystem:  true,
		},
		Privacy: Privacy{
			RecordWindowTitle: true,
			ExcludeApps:       []string{"1Password"},
		},
	}
}

// Load reads the config file at the default path, layering it over Default.
// A missing file is not an error; defaults are returned instead.
func Load() (Config, error) {
	return LoadFrom(Path())
}

// LoadFrom reads the config file at path, layering it over Default.
func LoadFrom(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.normalize()
	return cfg, nil
}

// normalize repairs values that would break the daemon if left at zero.
func (c *Config) normalize() {
	if c.Tracker.PollIntervalSeconds <= 0 {
		c.Tracker.PollIntervalSeconds = 5
	}
	if c.Tracker.IdleThresholdSeconds <= 0 {
		c.Tracker.IdleThresholdSeconds = 300
	}
	if c.Tracker.FocusBlockMinMinutes <= 0 {
		c.Tracker.FocusBlockMinMinutes = 15
	}
	if strings.TrimSpace(c.Storage.DBPath) == "" {
		c.Storage.DBPath = "~/.local/share/worklog/worklog.db"
	}
}

// ResolvedDBPath returns the database path with ~ expanded to the home dir.
func (c Config) ResolvedDBPath() string {
	return ExpandHome(c.Storage.DBPath)
}

// IsExcluded reports whether app is in the privacy exclusion list.
func (c Config) IsExcluded(app string) bool {
	for _, e := range c.Privacy.ExcludeApps {
		if strings.EqualFold(strings.TrimSpace(e), strings.TrimSpace(app)) {
			return true
		}
	}
	return false
}

// configDir returns ~/.config/worklog, honoring XDG_CONFIG_HOME.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "worklog")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "worklog")
	}
	return filepath.Join(home, ".config", "worklog")
}

// Path returns the path to config.toml.
func Path() string { return filepath.Join(configDir(), "config.toml") }

// RulesPath returns the path to rules.toml.
func RulesPath() string { return filepath.Join(configDir(), "rules.toml") }

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
