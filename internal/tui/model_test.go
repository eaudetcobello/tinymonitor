package tui_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/tui"
)

var errServer = errors.New("connection refused")

// call is one request made to a fakeSource.
type call struct{ from, to time.Time }

// fakeSource answers each Get with the next of its results, repeating the last one.
type fakeSource struct {
	mu      sync.Mutex
	calls   []call
	results []result
	// block, when set, holds each Get until it is closed.
	block chan struct{}
}

type result struct {
	metrics []model.Metric
	err     error
}

func (f *fakeSource) Get(_ context.Context, _ uint32, _ model.Kind, from, to time.Time) ([]model.Metric, error) {
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, call{from, to})

	r := f.results[min(len(f.calls), len(f.results))-1]

	return r.metrics, r.err
}

func (f *fakeSource) Calls() []call {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]call(nil), f.calls...)
}

// now is the fixed time of the tests' clock.
func now() time.Time {
	return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
}

func testConfig() tui.Config {
	return tui.Config{
		ServerURL: tui.DefaultServerURL,
		AgentID:   tui.DefaultAgentID,
		Kind:      model.KindMemory,
		Window:    tui.DefaultWindow,
		// A short refresh keeps the poll ticks of the tests fast.
		Refresh: time.Millisecond,
	}
}

// samples returns one memory sample per second, the last one a second before now, with used_pct set to values.
func samples(values ...float64) []model.Metric {
	out := make([]model.Metric, len(values))
	for i, v := range values {
		out[i] = model.Metric{
			AgentID: tui.DefaultAgentID,
			Kind:    model.KindMemory,
			Time:    now().Add(time.Duration(i-len(values)) * time.Second),
			Values:  map[string]float64{"used_pct": v, "avail_gib": 4},
		}
	}

	return out
}

func newModel(src tui.MetricsSource) tui.Model {
	return tui.New(testConfig(), src, tui.WithClock(now))
}

// update sends msg to m and returns the updated tui.Model.
func update(t *testing.T, m tui.Model, msg tea.Msg) (tui.Model, tea.Cmd) {
	t.Helper()

	next, cmd := m.Update(msg)

	got, ok := next.(tui.Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}

	return got, cmd
}

// poll runs cmd, a fetch, feeds its result to m and then runs the returned tick, so m is ready for the next fetch.
// It returns the model after the fetch and the command of the next fetch.
func poll(t *testing.T, m tui.Model, fetch tea.Cmd) (tui.Model, tea.Cmd) {
	t.Helper()

	m, tick := update(t, m, fetch())
	if tick == nil {
		t.Fatal("a fetch result did not schedule the next poll")
	}

	return update(t, m, tick())
}

// start sizes m to width x height and runs the first fetch.
func start(t *testing.T, src tui.MetricsSource, width, height int) (tui.Model, tea.Cmd) {
	t.Helper()

	m := newModel(src)
	m, _ = update(t, m, tea.WindowSizeMsg{Width: width, Height: height})

	return poll(t, m, m.Init())
}

func screen(m tui.Model) string {
	return ansi.Strip(m.View().Content)
}

func TestFirstFetchReadsTheWholeWindowThenOnlyNewSamples(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(1, 2, 3)}}}

	_, next := start(t, src, 80, 24)
	next()

	calls := src.Calls()
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}

	if want := (call{now().Add(-tui.DefaultWindow), now()}); calls[0] != want {
		t.Errorf("first call = %v, want %v", calls[0], want)
	}

	if want := now().Add(-time.Second); !calls[1].from.Equal(want) {
		t.Errorf("second call from = %v, want the newest sample %v", calls[1].from, want)
	}
}

func TestSlowFetchDoesNotBlockInput(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{}}, block: make(chan struct{})}
	defer close(src.block)

	m := newModel(src)
	fetch := m.Init()

	go fetch() // Bubble Tea runs commands in their own goroutines.

	done := make(chan tea.Cmd)

	go func() {
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
		done <- cmd
	}()

	select {
	case cmd := <-done:
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("q did not quit while a fetch was pending")
		}
	case <-time.After(time.Second):
		t.Fatal("Update blocked on a pending fetch")
	}
}

func TestGraphShowsLatestValueAndAxes(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20, 30, 25)}}}
	m, _ := start(t, src, 100, 30)

	out := screen(m)
	for _, want := range []string{
		"used_pct", "latest 25.0% (1s ago)", "min 10.0%", "max 30.0%", "over 5m0s",
		"31.0", "9.0", // Y bounds: min and max padded by 5% of the spread
		// X axis start, in the local time zone like the labels
		time.UnixMilli(now().Add(-tui.DefaultWindow).UnixMilli()).Format("15:04:05"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view does not contain %q:\n%s", want, out)
		}
	}

	if !strings.ContainsAny(out, "⠁⠂⠄⡀⠈⠐⠠⢀⣀⣿") {
		t.Errorf("view has no braille line:\n%s", out)
	}
}

func TestErrorKeepsLastDataAndRetries(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20, 30)}, {err: errServer}, {metrics: samples(40)}}}
	m, next := start(t, src, 100, 30)

	m, next = poll(t, m, next)

	out := screen(m)
	for _, want := range []string{"error: connection refused", "retrying every 1ms", "latest 30.0%"} {
		if !strings.Contains(out, want) {
			t.Errorf("after an error the view does not contain %q:\n%s", want, out)
		}
	}

	m, _ = poll(t, m, next)
	if out := screen(m); strings.Contains(out, "error:") {
		t.Errorf("the error stays after a successful fetch:\n%s", out)
	}
}

func TestPlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		res   result
		want  string
	}{
		{"empty result", "", result{metrics: []model.Metric{}}, "no data in the last 5m0s for agent 123123123 (memory)"},
		{"first fetch fails", "", result{err: errServer}, "no data yet, retrying"},
		{
			"value not in data", "swap_pct", result{metrics: samples(1)},
			`no "swap_pct" in the last 5m0s (available: avail_gib, used_pct)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testConfig()
			cfg.Value = tt.value
			m := tui.New(cfg, &fakeSource{results: []result{tt.res}}, tui.WithClock(now))
			m, _ = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 20})
			m, _ = poll(t, m, m.Init())

			if out := screen(m); !strings.Contains(out, tt.want) {
				t.Errorf("view does not contain %q:\n%s", tt.want, out)
			}
		})
	}
}

func TestViewFillsTerminalOnResize(t *testing.T) {
	t.Parallel()

	sizes := []struct{ width, height int }{{120, 40}, {80, 24}, {20, 5}, {5, 1}}

	states := map[string][]result{
		"loading": nil,
		"data":    {{metrics: samples(10, 20, 30)}},
		"error":   {{metrics: samples(10, 20, 30)}, {err: errServer}},
	}

	for name, results := range states {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			m := newModel(&fakeSource{results: results})
			for range results {
				m, _ = poll(t, m, m.Init())
			}

			for _, size := range sizes {
				m, _ = update(t, m, tea.WindowSizeMsg{Width: size.width, Height: size.height})

				v := m.View()
				if !v.AltScreen {
					t.Errorf("%dx%d: AltScreen is off", size.width, size.height)
				}

				if got := lipgloss.Height(v.Content); got != size.height {
					t.Errorf("%dx%d: view is %d lines high", size.width, size.height, got)
				}

				for i, line := range strings.Split(v.Content, "\n") {
					if got := lipgloss.Width(line); got != size.width {
						t.Errorf("%dx%d: line %d is %d cells wide", size.width, size.height, i, got)
					}
				}
			}
		})
	}
}

func TestViewBeforeFirstResizeUsesAltScreen(t *testing.T) {
	t.Parallel()

	if v := newModel(&fakeSource{}).View(); !v.AltScreen {
		t.Error("AltScreen is off before the first WindowSizeMsg")
	}
}

func TestQuitKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      tea.KeyPressMsg
		wantQuit bool
	}{
		{"q", tea.KeyPressMsg{Code: 'q', Text: "q"}, true},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, true},
		{"other key", tea.KeyPressMsg{Code: 'x', Text: "x"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, cmd := update(t, newModel(&fakeSource{}), tt.key)

			quit := false
			if cmd != nil {
				_, quit = cmd().(tea.QuitMsg)
			}

			if quit != tt.wantQuit {
				t.Errorf("quit = %v, want %v", quit, tt.wantQuit)
			}
		})
	}
}
