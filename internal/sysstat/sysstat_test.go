package sysstat

import "testing"

const sampleVMStat = `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               100000.
Pages active:                             200000.
Pages inactive:                            50000.
Pages speculative:                         10000.
Pages throttled:                               0.
Pages wired down:                         150000.
Pages purgeable:                           20000.
Pages occupied by compressor:              50000.
`

func TestParseVMStatUsed(t *testing.T) {
	// used = (active 200000 + wired 150000 + compressed 50000) * 16384
	got, err := parseVMStatUsed([]byte(sampleVMStat))
	if err != nil {
		t.Fatalf("parseVMStatUsed: %v", err)
	}
	want := uint64(400000) * 16384
	if got != want {
		t.Errorf("used = %d, want %d", got, want)
	}
}

func TestParseVMStatMissingFields(t *testing.T) {
	if _, err := parseVMStatUsed([]byte("Pages free: 1.\n")); err == nil {
		t.Error("expected error when active/wired are missing")
	}
}

func TestParseTopCPU(t *testing.T) {
	out := `Processes: 400 total
2026/05/29 21:00:00
CPU usage: 50.00% user, 30.00% sys, 20.00% idle
Load Avg: 2.0
CPU usage: 4.76% user, 9.52% sys, 85.71% idle
`
	got, err := parseTopCPU([]byte(out))
	if err != nil {
		t.Fatalf("parseTopCPU: %v", err)
	}
	// Should use the LAST sample: 100 - 85.71 = 14.29
	if got < 14.28 || got > 14.30 {
		t.Errorf("cpu used = %v, want ~14.29", got)
	}
}

func TestParseTopCPUNoLine(t *testing.T) {
	if _, err := parseTopCPU([]byte("nothing here\n")); err == nil {
		t.Error("expected error when no CPU usage line present")
	}
}

func TestParsePmset(t *testing.T) {
	tests := []struct {
		name        string
		out         string
		wantPresent bool
		wantPct     int
		wantState   string
	}{
		{
			name:        "discharging",
			out:         "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123)\t75%; discharging; 3:21 remaining present: true\n",
			wantPresent: true, wantPct: 75, wantState: "discharging",
		},
		{
			name:        "charged on AC",
			out:         "Now drawing from 'AC Power'\n -InternalBattery-0 (id=123)\t100%; charged; 0:00 remaining present: true\n",
			wantPresent: true, wantPct: 100, wantState: "charged",
		},
		{
			name:        "no battery (desktop)",
			out:         "Now drawing from 'AC Power'\n",
			wantPresent: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := parsePmset([]byte(tt.out))
			if err != nil {
				t.Fatalf("parsePmset: %v", err)
			}
			if b.Present != tt.wantPresent {
				t.Fatalf("present = %v, want %v", b.Present, tt.wantPresent)
			}
			if tt.wantPresent && (b.Percent != tt.wantPct || b.State != tt.wantState) {
				t.Errorf("got %d%% %q, want %d%% %q", b.Percent, b.State, tt.wantPct, tt.wantState)
			}
		})
	}
}

func TestParseMemFreePercent(t *testing.T) {
	out := "...\nSystem-wide memory free percentage: 74%\n...\n"
	got, ok := parseMemFreePercent([]byte(out))
	if !ok || got != 74 {
		t.Errorf("parseMemFreePercent = %d, %v; want 74, true", got, ok)
	}
	if _, ok := parseMemFreePercent([]byte("no such line\n")); ok {
		t.Error("expected ok=false when line absent")
	}
}

func TestPressureLevel(t *testing.T) {
	tests := []struct {
		free int
		want string
	}{
		{74, "low"}, {20, "medium"}, {5, "high"}, {-1, "unknown"},
	}
	for _, tt := range tests {
		if got := (MemStat{FreePercent: tt.free}).PressureLevel(); got != tt.want {
			t.Errorf("free=%d -> %q, want %q", tt.free, got, tt.want)
		}
	}
}

func TestMemStatUsedPercent(t *testing.T) {
	m := MemStat{TotalBytes: 1000, UsedBytes: 620}
	if p := m.UsedPercent(); p != 62 {
		t.Errorf("UsedPercent = %v, want 62", p)
	}
	if p := (MemStat{}).UsedPercent(); p != 0 {
		t.Errorf("zero total should give 0, got %v", p)
	}
}
