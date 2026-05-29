// Package tracker observes the foreground application, window title and idle
// time on macOS and writes events to the store on a fixed interval.
//
// All macOS access goes through subprocesses (osascript, ioreg) to keep the
// build cgo-free. The functions are exposed as package variables so tests can
// substitute fakes.
package tracker

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Sample is one observation of the system state.
type Sample struct {
	App     string
	Title   string
	IdleSec int
}

// activeAppFn returns the name of the frontmost application.
var activeAppFn = func() (string, error) {
	out, err := exec.Command("osascript", "-e",
		`tell application "System Events" to get name of first process whose frontmost is true`,
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// activeTitleFn returns the title of the frontmost window, or "" if none.
var activeTitleFn = func() (string, error) {
	const script = `
tell application "System Events"
  set frontApp to first process whose frontmost is true
  try
    return name of front window of frontApp
  on error
    return ""
  end try
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var hidIdleRE = regexp.MustCompile(`"HIDIdleTime"\s*=\s*(\d+)`)

// idleSecondsFn returns how many seconds the user has been idle.
var idleSecondsFn = func() (int, error) {
	out, err := exec.Command("ioreg", "-c", "IOHIDSystem").Output()
	if err != nil {
		return 0, err
	}
	m := hidIdleRE.FindSubmatch(out)
	if len(m) < 2 {
		return 0, errors.New("HIDIdleTime not found")
	}
	ns, err := strconv.ParseInt(string(m[1]), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(ns / 1e9), nil
}

// Probe takes a single sample of the current system state.
func Probe() (Sample, error) {
	app, err := activeAppFn()
	if err != nil {
		return Sample{}, err
	}
	title, err := activeTitleFn()
	if err != nil {
		// A missing window title is common (no front window); don't fail.
		title = ""
	}
	idle, err := idleSecondsFn()
	if err != nil {
		idle = 0
	}
	return Sample{App: app, Title: title, IdleSec: idle}, nil
}
