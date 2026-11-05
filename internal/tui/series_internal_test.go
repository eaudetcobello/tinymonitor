package tui

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
)

// base is the reference time of the test samples.
func base() time.Time {
	return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
}

// at returns base plus n seconds.
func at(n float64) time.Time {
	return base().Add(time.Duration(n * float64(time.Second)))
}

func metric(sec float64, values map[string]float64) model.Metric {
	return model.Metric{AgentID: 1, Kind: model.KindMemory, Time: at(sec), Values: values}
}

func times(metrics []model.Metric) []time.Time {
	out := make([]time.Time, len(metrics))
	for i, m := range metrics {
		out[i] = m.Time
	}

	return out
}

func pts(pairs ...float64) []point {
	out := make([]point, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, point{t: at(pairs[i]), v: pairs[i+1]})
	}

	return out
}

func TestMerge(t *testing.T) {
	t.Parallel()

	v := map[string]float64{"used_pct": 1}

	tests := []struct {
		name       string
		buf, fresh []model.Metric
		from       time.Time
		want       []time.Time
	}{
		{"empty buffer takes everything", nil, []model.Metric{metric(1, v), metric(2, v)}, at(0), []time.Time{at(1), at(2)}},
		{
			"boundary sample is not duplicated",
			[]model.Metric{metric(1, v), metric(2, v)},
			[]model.Metric{metric(2, v), metric(3, v)},
			at(0),
			[]time.Time{at(1), at(2), at(3)},
		},
		{
			"samples before from are dropped",
			[]model.Metric{metric(1, v), metric(2, v)},
			[]model.Metric{metric(3, v)},
			at(2),
			[]time.Time{at(2), at(3)},
		},
		{"empty fresh keeps buffer", []model.Metric{metric(1, v)}, nil, at(0), []time.Time{at(1)}},
		{"everything too old", []model.Metric{metric(1, v)}, []model.Metric{metric(2, v)}, at(5), []time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := times(merge(tt.buf, tt.fresh, tt.from)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("merge times = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectValue(t *testing.T) {
	t.Parallel()

	data := []model.Metric{metric(1, map[string]float64{"b": 1, "a": 2}), metric(2, map[string]float64{"c": 3})}

	tests := []struct {
		name      string
		want      string
		kind      model.Kind
		metrics   []model.Metric
		wantValue string
	}{
		{"explicit value wins", "c", model.KindMemory, data, "c"},
		{"kind default", "", model.KindMemory, data, "used_pct"},
		{"first name without a default", "", model.KindCPU, data, "a"},
		{"nothing to pick", "", model.KindCPU, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value, available := selectValue(tt.want, tt.kind, tt.metrics)
			if value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}

			if len(tt.metrics) > 0 && !reflect.DeepEqual(available, []string{"a", "b", "c"}) {
				t.Errorf("available = %v, want [a b c]", available)
			}
		})
	}
}

func TestSeriesFor(t *testing.T) {
	t.Parallel()

	metrics := []model.Metric{
		metric(0, map[string]float64{"x": 1}),           // before from
		metric(1, map[string]float64{"x": 2}),           // == from
		metric(2, map[string]float64{"y": 9}),           // other value only
		metric(3, map[string]float64{"x": math.NaN()}),  // not finite
		metric(4, map[string]float64{"x": math.Inf(1)}), // not finite
		metric(5, map[string]float64{"x": 3, "y": 8}),   // == to
		metric(6, map[string]float64{"x": 4}),           // after to
	}

	got := seriesFor(metrics, "x", at(1), at(5))
	if want := pts(1, 2, 5, 3); !reflect.DeepEqual(got, want) {
		t.Errorf("seriesFor = %v, want %v", got, want)
	}
}

func TestSplitRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		points []point
		want   [][]point
	}{
		{"empty", nil, nil},
		{"one run", pts(0, 1, 1, 2, 2, 3), [][]point{pts(0, 1, 1, 2, 2, 3)}},
		{"gap splits", pts(0, 1, 1, 2, 10, 3, 11, 4), [][]point{pts(0, 1, 1, 2), pts(10, 3, 11, 4)}},
		{"spacing equal to max is joined", pts(0, 1, 2, 2), [][]point{pts(0, 1, 2, 2)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := splitRuns(tt.points, 2*time.Second); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitRuns = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaxGap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		points []point
		bucket time.Duration
		want   time.Duration
	}{
		{"single point uses two buckets", pts(0, 1), time.Second, 2 * time.Second},
		{"median spacing times four", pts(0, 1, 0.5, 1, 1, 1, 1.5, 1, 60, 1), 100 * time.Millisecond, 2 * time.Second},
		{"wide buckets win", pts(0, 1, 0.5, 1, 1, 1), 5 * time.Second, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := maxGap(tt.points, tt.bucket); got != tt.want {
				t.Errorf("maxGap = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownsample(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		points  []point
		buckets int
		want    []point
	}{
		{"fewer points than buckets are unchanged", pts(1, 5, 7, 6), 10, pts(1, 5, 7, 6)},
		// [0,10] in 2 buckets: [0,5) and [5,10].
		{"means per bucket", pts(0, 1, 2, 3, 4, 5, 6, 10, 10, 20), 2, pts(2, 3, 8, 15)},
		{"empty buckets emit nothing", pts(0, 1, 9, 2), 10, pts(0, 1, 9, 2)},
		{"points outside the range are dropped", pts(-1, 100, 5, 1, 11, 100), 4, pts(5, 1)},
		{"zero buckets", pts(1, 1), 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := downsample(tt.points, at(0), at(10), tt.buckets)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("downsample = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownsampleNeverExceedsBuckets(t *testing.T) {
	t.Parallel()

	// 10 minutes at 2 samples per second into 80 columns of braille (160 dots).
	points := make([]point, 0, 1200)
	for i := range 1200 {
		points = append(points, point{t: base().Add(time.Duration(i) * 500 * time.Millisecond), v: float64(i % 7)})
	}

	got := downsample(points, base(), base().Add(10*time.Minute), 160)
	if len(got) != 160 {
		t.Fatalf("len = %d, want 160", len(got))
	}

	for i := 1; i < len(got); i++ {
		if !got[i].t.After(got[i-1].t) {
			t.Fatalf("point %d at %v is not after %v", i, got[i].t, got[i-1].t)
		}
	}
}

func TestBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		points         []point
		wantLo, wantHi float64
		wantOK         bool
	}{
		{"empty", nil, 0, 0, false},
		{"single point", pts(0, 50), 49, 51, true},
		{"flat", pts(0, 3, 1, 3), 2, 4, true},
		{"padded by 5%", pts(0, 10, 1, 30, 2, 20), 9, 31, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lo, hi, ok := bounds(tt.points)
			if lo != tt.wantLo || hi != tt.wantHi || ok != tt.wantOK {
				t.Errorf("bounds = (%v, %v, %v), want (%v, %v, %v)", lo, hi, ok, tt.wantLo, tt.wantHi, tt.wantOK)
			}
		})
	}
}
