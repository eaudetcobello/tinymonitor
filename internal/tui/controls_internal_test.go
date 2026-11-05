package tui

import (
	"testing"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const availGiB = "avail_gib"

func TestWiderWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cur, want time.Duration
	}{
		{time.Minute, 5 * time.Minute},
		{5 * time.Minute, 15 * time.Minute},
		{15 * time.Minute, time.Hour},
		{time.Hour, time.Hour},               // longest preset
		{10 * time.Minute, 15 * time.Minute}, // between presets
		{30 * time.Second, time.Minute},      // below the presets
		{3 * time.Hour, 3 * time.Hour},       // above the presets
	}

	for _, tt := range tests {
		if got := widerWindow(tt.cur); got != tt.want {
			t.Errorf("widerWindow(%s) = %s, want %s", tt.cur, got, tt.want)
		}
	}
}

func TestNarrowerWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cur, refresh, want time.Duration
	}{
		{time.Hour, time.Second, 15 * time.Minute},
		{5 * time.Minute, time.Second, time.Minute},
		{time.Minute, time.Second, time.Minute},             // shortest preset
		{10 * time.Minute, time.Second, 5 * time.Minute},    // between presets
		{3 * time.Hour, time.Second, time.Hour},             // above the presets
		{5 * time.Minute, 2 * time.Minute, 5 * time.Minute}, // 1m is shorter than the refresh
		{time.Hour, 2 * time.Minute, 15 * time.Minute},
		{5 * time.Minute, time.Minute, time.Minute}, // a window equal to the refresh is allowed
	}

	for _, tt := range tests {
		if got := narrowerWindow(tt.cur, tt.refresh); got != tt.want {
			t.Errorf("narrowerWindow(%s, %s) = %s, want %s", tt.cur, tt.refresh, got, tt.want)
		}
	}
}

func TestNextValue(t *testing.T) {
	t.Parallel()

	available := []string{availGiB, "total_gib", usedPct}

	tests := []struct {
		name      string
		cur       string
		available []string
		want      string
	}{
		{"next name", availGiB, available, "total_gib"},
		{"wraps around", usedPct, available, availGiB},
		{"unknown value goes to the first", "swap_pct", available, availGiB},
		{"nothing available keeps the value", usedPct, nil, usedPct},
	}

	for _, tt := range tests {
		if got := nextValue(tt.cur, tt.available); got != tt.want {
			t.Errorf("%s: nextValue(%q) = %q, want %q", tt.name, tt.cur, got, tt.want)
		}
	}
}

func TestNextKind(t *testing.T) {
	t.Parallel()

	if got := nextKind(model.KindCPU); got != model.KindMemory {
		t.Errorf("nextKind(cpu) = %s, want memory", got)
	}

	if got := nextKind(model.KindMemory); got != model.KindCPU {
		t.Errorf("nextKind(memory) = %s, want cpu", got)
	}
}
