package collector

import (
	"errors"
	"math"
	"testing"

	"github.com/shirou/gopsutil/v4/cpu"
)

func TestCPUUsage(t *testing.T) {
	t.Parallel()

	prev := cpu.TimesStat{User: 100, System: 50, Idle: 800, Iowait: 50, Nice: 0}

	tests := []struct {
		name string
		cur  cpu.TimesStat
		want map[string]float64
	}{
		{
			name: "busy share of the elapsed time",
			// +100 total: 30 user, 10 system, 50 idle, 10 iowait.
			cur:  cpu.TimesStat{User: 130, System: 60, Idle: 850, Iowait: 60},
			want: map[string]float64{usedPct: 40, userPct: 30, systemPct: 10},
		},
		{
			name: "fully idle",
			cur:  cpu.TimesStat{User: 100, System: 50, Idle: 900, Iowait: 50},
			want: map[string]float64{usedPct: 0, userPct: 0, systemPct: 0},
		},
		{
			name: "guest time is part of user time, not extra total",
			// +100 total, all user, half of it guest.
			cur:  cpu.TimesStat{User: 200, System: 50, Idle: 800, Iowait: 50, Guest: 50},
			want: map[string]float64{usedPct: 100, userPct: 100, systemPct: 0},
		},
		{
			name: "counter jitter is clamped",
			// Idle grew more than the total did.
			cur:  cpu.TimesStat{User: 90, System: 50, Idle: 820, Iowait: 50},
			want: map[string]float64{usedPct: 0, userPct: 0, systemPct: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cpuUsage(prev, tt.cur)
			if err != nil {
				t.Fatalf("cpuUsage: %v", err)
			}

			for name, want := range tt.want {
				if math.Abs(got[name]-want) > 1e-9 {
					t.Errorf("%s = %v, want %v", name, got[name], want)
				}
			}

			if len(got) != len(tt.want) {
				t.Errorf("values = %v, want the keys of %v", got, tt.want)
			}
		})
	}
}

func TestCPUUsageWithoutElapsedTimeFails(t *testing.T) {
	t.Parallel()

	same := cpu.TimesStat{User: 1, Idle: 1}
	if _, err := cpuUsage(same, same); err == nil {
		t.Fatal("cpuUsage of the same reading: want an error")
	}
}

func TestGopsutilCPUNeedsABaseline(t *testing.T) {
	t.Parallel()

	var g Gopsutil

	if _, err := g.CPU(); !errors.Is(err, ErrNoBaseline) {
		t.Fatalf("first CPU() error = %v, want ErrNoBaseline", err)
	}

	// The second reading may come too soon for the counters to move; then it errors instead of reporting 0%.
	m, err := g.CPU()
	if err != nil {
		if errors.Is(err, errNoCPUTime) {
			t.Skip("cpu counters did not advance between readings")
		}

		t.Fatalf("second CPU(): %v", err)
	}

	for _, name := range []string{usedPct, userPct, systemPct} {
		if v, ok := m.Values[name]; !ok || v < 0 || v > 100 {
			t.Errorf("%s = %v (present %v), want a value in [0, 100]", name, v, ok)
		}
	}
}
