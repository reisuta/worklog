package format

import "testing"

func TestDuration(t *testing.T) {
	tests := []struct {
		name string
		sec  int
		want string
	}{
		{"zero", 0, "0s"},
		{"negative clamped", -10, "0s"},
		{"seconds only", 45, "45s"},
		{"minutes", 23 * 60, "23m"},
		{"minutes drop seconds", 23*60 + 12, "23m"},
		{"hours and minutes", 4*3600 + 23*60, "4h 23m"},
		{"exact hour", 3600, "1h 0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Duration(tt.sec); got != tt.want {
				t.Errorf("Duration(%d) = %q, want %q", tt.sec, got, tt.want)
			}
		})
	}
}

func TestBar(t *testing.T) {
	tests := []struct {
		name              string
		value, max, width int
		want              string
	}{
		{"zero width", 5, 10, 0, ""},
		{"empty when max zero", 5, 0, 4, "░░░░"},
		{"half", 5, 10, 4, "██░░"},
		{"full", 10, 10, 4, "████"},
		{"overflow clamps", 20, 10, 4, "████"},
		{"none", 0, 10, 4, "░░░░"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Bar(tt.value, tt.max, tt.width); got != tt.want {
				t.Errorf("Bar(%d,%d,%d) = %q, want %q", tt.value, tt.max, tt.width, got, tt.want)
			}
		})
	}
}
