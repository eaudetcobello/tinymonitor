package tui

import (
	"math"
	"slices"
	"sort"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
)

// gapSpacings is how many typical sample spacings two samples may be apart before the line breaks.
const gapSpacings = 4

// defaultValue names the series graphed for kind when -value is empty, or "" when the kind has none.
func defaultValue(kind model.Kind) string {
	switch kind {
	case model.KindMemory:
		return "used_pct"
	case model.KindCPU:
		return ""
	}

	return ""
}

// point is one sample of a single value.
type point struct {
	t time.Time
	v float64
}

// merge appends the samples of fresh that are newer than the last sample of buf, then drops samples before from.
// The API range is inclusive, so a poll starting at the last buffered time returns that sample again.
// Both inputs are ordered by time; the result is too.
func merge(buf, fresh []model.Metric, from time.Time) []model.Metric {
	for _, m := range fresh {
		if len(buf) == 0 || m.Time.After(buf[len(buf)-1].Time) {
			buf = append(buf, m)
		}
	}

	i := sort.Search(len(buf), func(i int) bool { return !buf[i].Time.Before(from) })

	return slices.Clip(buf[i:])
}

// selectValue returns the value to graph (want when set, else the kind's default, else the first name in metrics)
// and every value name in metrics, sorted.
func selectValue(want string, kind model.Kind, metrics []model.Metric) (string, []string) {
	seen := map[string]bool{}

	var available []string

	for _, m := range metrics {
		for name := range m.Values {
			if !seen[name] {
				seen[name] = true

				available = append(available, name)
			}
		}
	}

	sort.Strings(available)

	switch {
	case want != "":
		return want, available
	case defaultValue(kind) != "":
		return defaultValue(kind), available
	case len(available) > 0:
		return available[0], available
	default:
		return "", available
	}
}

// seriesFor returns the finite samples of value with a time in [from, to].
func seriesFor(metrics []model.Metric, value string, from, to time.Time) []point {
	var points []point

	for _, m := range metrics {
		if m.Time.Before(from) || m.Time.After(to) {
			continue
		}

		v, ok := m.Values[value]
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}

		points = append(points, point{t: m.Time, v: v})
	}

	return points
}

// splitRuns cuts points wherever two neighbours are more than maxGap apart, so missing data shows as a gap.
func splitRuns(points []point, maxGap time.Duration) [][]point {
	var runs [][]point

	start := 0

	for i := 1; i <= len(points); i++ {
		if i == len(points) || points[i].t.Sub(points[i-1].t) > maxGap {
			runs = append(runs, points[start:i])
			start = i
		}
	}

	return runs
}

// maxGap returns the spacing above which two samples are not joined by a line:
// gapSpacings times the median sample spacing, and at least two buckets so downsampling never opens a gap.
func maxGap(points []point, bucket time.Duration) time.Duration {
	floor := 2 * bucket
	if len(points) < 2 {
		return floor
	}

	spacings := make([]time.Duration, len(points)-1)
	for i := 1; i < len(points); i++ {
		spacings[i-1] = points[i].t.Sub(points[i-1].t)
	}

	slices.Sort(spacings)

	return max(floor, gapSpacings*spacings[len(spacings)/2])
}

// downsample splits [from, to] into buckets equal time slices and returns one point per non-empty slice:
// the mean time and mean value of its samples. Points outside [from, to] are dropped.
func downsample(points []point, from, to time.Time, buckets int) []point {
	span := to.Sub(from)
	if buckets <= 0 || span <= 0 {
		return nil
	}

	type acc struct {
		t, v float64
		n    int
	}

	accs := make([]acc, buckets)

	for _, p := range points {
		if p.t.Before(from) || p.t.After(to) {
			continue
		}

		i := min(int(float64(p.t.Sub(from))/float64(span)*float64(buckets)), buckets-1)
		accs[i].t += float64(p.t.Sub(from))
		accs[i].v += p.v
		accs[i].n++
	}

	var out []point

	for _, a := range accs {
		if a.n > 0 {
			n := float64(a.n)
			out = append(out, point{t: from.Add(time.Duration(a.t / n)), v: a.v / n})
		}
	}

	return out
}

// bounds returns the Y range (low, high) for points: their min and max padded by 5% of the spread.
// A flat series gets ±1 so its line is not drawn on an axis edge. The bool is false when points is empty.
func bounds(points []point) (float64, float64, bool) {
	if len(points) == 0 {
		return 0, 0, false
	}

	lo, hi := points[0].v, points[0].v
	for _, p := range points[1:] {
		lo, hi = min(lo, p.v), max(hi, p.v)
	}

	if hi == lo {
		return lo - 1, hi + 1, true
	}

	pad := (hi - lo) * 0.05

	return lo - pad, hi + pad, true
}
