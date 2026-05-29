// Package sysstat reads basic system resource usage (memory, CPU, battery) on
// macOS via subprocesses, keeping worklog cgo-free in the same spirit as the
// osascript / ioreg wrappers in package tracker.
package sysstat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MemStat describes physical memory usage.
type MemStat struct {
	TotalBytes uint64
	UsedBytes  uint64
	// FreePercent is the system-wide memory free percentage reported by
	// memory_pressure (the basis of macOS's green/yellow/red indicator), or
	// -1 if it could not be read.
	FreePercent int
}

// UsedPercent returns memory in use as a percentage of total.
func (m MemStat) UsedPercent() float64 {
	if m.TotalBytes == 0 {
		return 0
	}
	return float64(m.UsedBytes) / float64(m.TotalBytes) * 100
}

// PressureLevel classifies memory pressure from FreePercent, mirroring
// Activity Monitor's green/yellow/red. It reflects how tight memory really is
// (cache filling RAM stays "low"), not raw usage.
func (m MemStat) PressureLevel() string {
	switch {
	case m.FreePercent < 0:
		return "unknown"
	case m.FreePercent < 10:
		return "high"
	case m.FreePercent < 25:
		return "medium"
	default:
		return "low"
	}
}

// BatteryStat describes the battery state. Present is false on machines that
// have no battery (e.g. a Mac desktop).
type BatteryStat struct {
	Present bool
	Percent int
	State   string // "charging", "discharging", "charged", or ""
}

// Snapshot is a best-effort collection of metrics. A nil field means that
// metric could not be read (so one failing reader never blanks the others).
type Snapshot struct {
	Memory  *MemStat
	CPU     *float64
	Battery *BatteryStat
}

// Command runners, kept as package vars so tests can substitute fixtures.
var (
	runVMStat      = func() ([]byte, error) { return exec.Command("vm_stat").Output() }
	runMemsize     = func() ([]byte, error) { return exec.Command("sysctl", "-n", "hw.memsize").Output() }
	runTop         = func() ([]byte, error) { return exec.Command("top", "-l", "2", "-n", "0").Output() }
	runPmset       = func() ([]byte, error) { return exec.Command("pmset", "-g", "batt").Output() }
	runMemPressure = func() ([]byte, error) { return exec.Command("memory_pressure").Output() }
)

// Collect gathers every metric on a best-effort basis.
func Collect() Snapshot {
	var s Snapshot
	if m, err := Memory(); err == nil {
		s.Memory = &m
	}
	if c, err := CPUPercent(); err == nil {
		s.CPU = &c
	}
	if b, err := Battery(); err == nil && b.Present {
		s.Battery = &b
	}
	return s
}

// Memory reports physical memory usage. "Used" is the memory in active use:
// (active + wired + compressed) pages, which tracks Activity Monitor's
// "Memory Used" reasonably while excluding reclaimable cache.
func Memory() (MemStat, error) {
	totalRaw, err := runMemsize()
	if err != nil {
		return MemStat{}, fmt.Errorf("read hw.memsize: %w", err)
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(totalRaw)), 10, 64)
	if err != nil {
		return MemStat{}, fmt.Errorf("parse hw.memsize: %w", err)
	}
	vm, err := runVMStat()
	if err != nil {
		return MemStat{}, fmt.Errorf("run vm_stat: %w", err)
	}
	used, err := parseVMStatUsed(vm)
	if err != nil {
		return MemStat{}, err
	}
	m := MemStat{TotalBytes: total, UsedBytes: used, FreePercent: -1}
	// Memory pressure is best-effort; usage is still useful without it.
	if out, err := runMemPressure(); err == nil {
		if free, ok := parseMemFreePercent(out); ok {
			m.FreePercent = free
		}
	}
	return m, nil
}

// parseMemFreePercent extracts NN from memory_pressure's
// "System-wide memory free percentage: NN%".
func parseMemFreePercent(out []byte) (int, bool) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "free percentage") {
			continue
		}
		if i := strings.IndexByte(line, ':'); i >= 0 {
			if n, ok := firstUint(line[i:]); ok {
				return int(n), true
			}
		}
	}
	return 0, false
}

func parseVMStatUsed(out []byte) (uint64, error) {
	pageSize := uint64(4096)
	var active, wired, compressed uint64
	var haveActive, haveWired bool

	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "page size of") {
			if n, ok := firstUint(line); ok {
				pageSize = n
			}
			continue
		}
		key, val, ok := splitVMStatLine(line)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(key, "Pages active"):
			active, haveActive = val, true
		case strings.HasPrefix(key, "Pages wired down"):
			wired, haveWired = val, true
		case strings.HasPrefix(key, "Pages occupied by compressor"):
			compressed = val
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if !haveActive || !haveWired {
		return 0, errors.New("vm_stat: missing active/wired fields")
	}
	return (active + wired + compressed) * pageSize, nil
}

// splitVMStatLine parses "Pages active:    234567." into ("Pages active", 234567).
func splitVMStatLine(line string) (key string, val uint64, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", 0, false
	}
	key = strings.TrimSpace(line[:idx])
	num := strings.TrimSuffix(strings.TrimSpace(line[idx+1:]), ".")
	n, err := strconv.ParseUint(num, 10, 64)
	if err != nil {
		return "", 0, false
	}
	return key, n, true
}

// firstUint returns the first run of digits in s as a uint.
func firstUint(s string) (uint64, bool) {
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			n, err := strconv.ParseUint(s[start:i], 10, 64)
			return n, err == nil
		}
	}
	if start >= 0 {
		n, err := strconv.ParseUint(s[start:], 10, 64)
		return n, err == nil
	}
	return 0, false
}

// CPUPercent reports overall CPU usage (100 - idle) from the most recent
// `top` sample.
func CPUPercent() (float64, error) {
	out, err := runTop()
	if err != nil {
		return 0, fmt.Errorf("run top: %w", err)
	}
	return parseTopCPU(out)
}

func parseTopCPU(out []byte) (float64, error) {
	// `top -l 2` prints two "CPU usage:" lines; the second covers the sample
	// interval (the first is averaged since boot), so we take the last one.
	var last string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		if line := sc.Text(); strings.HasPrefix(line, "CPU usage:") {
			last = line
		}
	}
	if last == "" {
		return 0, errors.New("top: no CPU usage line")
	}
	// "CPU usage: 4.76% user, 9.52% sys, 85.71% idle"
	i := strings.LastIndexByte(last, ',')
	if i < 0 {
		return 0, errors.New("top: unexpected CPU usage format")
	}
	idlePart := strings.TrimSpace(last[i+1:])
	idlePart = strings.TrimSuffix(idlePart, "idle")
	idlePart = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(idlePart), "%"))
	idle, err := strconv.ParseFloat(idlePart, 64)
	if err != nil {
		return 0, fmt.Errorf("parse idle: %w", err)
	}
	used := 100 - idle
	if used < 0 {
		used = 0
	}
	return used, nil
}

// Battery reports the battery percentage and charging state. Present is false
// when no battery is found.
func Battery() (BatteryStat, error) {
	out, err := runPmset()
	if err != nil {
		return BatteryStat{}, fmt.Errorf("run pmset: %w", err)
	}
	return parsePmset(out)
}

func parsePmset(out []byte) (BatteryStat, error) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, '%')
		if i < 0 {
			continue
		}
		j := i
		for j > 0 && line[j-1] >= '0' && line[j-1] <= '9' {
			j--
		}
		if j == i {
			continue
		}
		pct, err := strconv.Atoi(line[j:i])
		if err != nil {
			continue
		}
		// "80%; discharging; 3:21 remaining present: true"
		rest := strings.TrimSpace(strings.TrimPrefix(line[i+1:], ";"))
		state := rest
		if k := strings.IndexByte(rest, ';'); k >= 0 {
			state = strings.TrimSpace(rest[:k])
		}
		return BatteryStat{Present: true, Percent: pct, State: state}, nil
	}
	return BatteryStat{Present: false}, nil
}
