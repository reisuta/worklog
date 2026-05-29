// Package daemon manages worklog's background process: the PID file, forking
// into the background, and macOS launchd integration.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/reisuta/worklog/internal/config"
)

// PIDPath returns the path to the PID file, alongside the database.
func PIDPath(cfg config.Config) string {
	return filepath.Join(filepath.Dir(cfg.ResolvedDBPath()), "worklog.pid")
}

// LogPath returns the path to the daemon log file.
func LogPath(cfg config.Config) string {
	return filepath.Join(filepath.Dir(cfg.ResolvedDBPath()), "worklog.log")
}

// writePID writes the current process PID to path.
func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

// readPID reads the PID from path. It returns 0 (no error) if the file is
// missing.
func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file %s: %w", path, err)
	}
	return pid, nil
}

// processAlive reports whether a process with pid is running, using the
// signal-0 trick (no signal sent, just an existence/permission check).
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
