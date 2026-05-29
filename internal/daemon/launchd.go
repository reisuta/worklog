package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/reisuta/worklog/internal/config"
)

// launchdLabel is the launchd job label.
const launchdLabel = "com.reisuta.worklog"

// plistTemplate is the launchd user-agent plist. %s placeholders are, in
// order: label, executable path, log path, log path.
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>daemon</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`

// plistPath returns ~/Library/LaunchAgents/com.reisuta.worklog.plist.
func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"), nil
}

// InstallLaunchd writes the launchd plist and loads it so worklog starts at
// login. It returns the plist path on success.
func InstallLaunchd(cfg config.Config) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	path, err := plistPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	logPath := LogPath(cfg)
	content := fmt.Sprintf(plistTemplate, launchdLabel, exe, logPath, logPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write plist: %w", err)
	}
	// Reload to pick up any changes; ignore unload errors (may not be loaded).
	_ = exec.Command("launchctl", "unload", path).Run()
	if out, err := exec.Command("launchctl", "load", path).CombinedOutput(); err != nil {
		return path, fmt.Errorf("launchctl load: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return path, nil
}

// UninstallLaunchd unloads and removes the launchd plist.
func UninstallLaunchd() (string, error) {
	path, err := plistPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, fmt.Errorf("not installed: %s", path)
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	if err := os.Remove(path); err != nil {
		return path, fmt.Errorf("remove plist: %w", err)
	}
	return path, nil
}
