package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/reisuta/worklog/internal/config"
)

// Status describes whether the daemon is running.
type Status struct {
	Running bool
	PID     int
}

// Query reports the current daemon status from the PID file.
func Query(cfg config.Config) (Status, error) {
	pid, err := readPID(PIDPath(cfg))
	if err != nil {
		return Status{}, err
	}
	if pid == 0 || !processAlive(pid) {
		return Status{Running: false}, nil
	}
	return Status{Running: true, PID: pid}, nil
}

// Start forks a detached background process running the foreground daemon
// (`worklog daemon`) and records its PID. It returns an error if the daemon is
// already running.
func Start(cfg config.Config) (int, error) {
	if st, err := Query(cfg); err != nil {
		return 0, err
	} else if st.Running {
		return st.PID, errors.New("worklog is already running")
	}

	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("locate executable: %w", err)
	}

	logFile, err := os.OpenFile(LogPath(cfg), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// Detach into a new session so the daemon outlives the launching shell.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start daemon: %w", err)
	}

	pid := cmd.Process.Pid
	if err := writePID(PIDPath(cfg), pid); err != nil {
		return pid, fmt.Errorf("write pid: %w", err)
	}
	// Don't reap the child; let it run independently.
	_ = cmd.Process.Release()
	return pid, nil
}

// Stop signals the running daemon to terminate and waits briefly for it to
// exit, then removes the PID file.
func Stop(cfg config.Config) error {
	pidPath := PIDPath(cfg)
	pid, err := readPID(pidPath)
	if err != nil {
		return err
	}
	if pid == 0 || !processAlive(pid) {
		_ = os.Remove(pidPath)
		return errors.New("worklog is not running")
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal daemon: %w", err)
	}
	for i := 0; i < 50; i++ { // up to ~5s
		if !processAlive(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = os.Remove(pidPath)
	return nil
}
