package tui_test

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/tui"
)

// errServer is what the client returns when the server is down: the request URL wrapping the dial error.
var errServer = &url.Error{Op: "Get", URL: "http://localhost:1323/metrics", Err: syscall.ECONNREFUSED}

// call is one request made to a fakeSource.
type call struct {
	kind     model.Kind
	from, to time.Time
}

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

// Value names the tests use.
const (
	usedPct  = "used_pct"
	availGiB = "avail_gib"
)

func (f *fakeSource) Get(_ context.Context, _ uint32, kind model.Kind, from, to time.Time) ([]model.Metric, error) {
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, call{kind, from, to})

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
			Values:  map[string]float64{usedPct: v, availGiB: 4},
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

	if want := (call{model.KindMemory, now().Add(-tui.DefaultWindow), now()}); calls[0] != want {
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
	for _, want := range []string{"error (retrying every 1ms): connection refused", "latest 30.0%"} {
		if !strings.Contains(out, want) {
			t.Errorf("after an error the view does not contain %q:\n%s", want, out)
		}
	}

	m, _ = poll(t, m, next)
	if out := screen(m); strings.Contains(out, "error") {
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

// key returns the press of the key that String() reports as name.
func key(name string) tea.KeyPressMsg {
	if name == "space" {
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}

	r := []rune(name)[0]

	return tea.KeyPressMsg{Code: r, Text: name}
}

// cpuSamples returns one cpu sample per second, the last one a second before now, with used_pct set to values.
func cpuSamples(values ...float64) []model.Metric {
	out := samples(values...)
	for i, v := range values {
		out[i].Kind = model.KindCPU
		out[i].Values = map[string]float64{usedPct: v, "user_pct": 1, "system_pct": 2}
	}

	return out
}

// lastCall returns the newest request made to src.
func lastCall(t *testing.T, src *fakeSource) call {
	t.Helper()

	calls := src.Calls()
	if len(calls) == 0 {
		t.Fatal("no request was made")
	}

	return calls[len(calls)-1]
}

func assertContains(t *testing.T, m tui.Model, wants ...string) {
	t.Helper()

	out := screen(m)
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("view does not contain %q:\n%s", want, out)
		}
	}
}

func TestFooterListsKeys(t *testing.T) {
	t.Parallel()

	m, _ := start(t, &fakeSource{results: []result{{metrics: samples(1)}}}, 100, 20)
	assertContains(t, m, "k kind  v value  +/- window  space pause  q quit")
}

func TestKindKeySwitchesKindAndDropsStaleResults(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{
		{metrics: samples(10, 20)}, {metrics: samples(30)}, {metrics: cpuSamples(70, 80)},
	}}
	m, stale := start(t, src, 100, 20)

	m, fetch := update(t, m, key("k"))
	if fetch == nil {
		t.Fatal("k did not start a fetch")
	}

	assertContains(t, m, "  cpu  window 5m0s", "loading…")

	// The memory poll that was in flight answers after the switch: it must not land or start another chain.
	m, cmd := update(t, m, stale())
	if cmd != nil {
		t.Error("a result from before the switch scheduled a poll")
	}

	assertContains(t, m, "loading…")

	m, cmd = update(t, m, fetch())
	if cmd == nil {
		t.Error("the cpu result did not schedule the next poll")
	}

	if want := (call{model.KindCPU, now().Add(-tui.DefaultWindow), now()}); lastCall(t, src) != want {
		t.Errorf("cpu request = %v, want %v", lastCall(t, src), want)
	}

	assertContains(t, m, "used_pct  latest 80.0%")

	m, fetch = update(t, m, key("k"))
	m, _ = update(t, m, fetch())

	if got := lastCall(t, src).kind; got != model.KindMemory {
		t.Errorf("second k requested %s, want memory", got)
	}

	assertContains(t, m, "  memory  window")
}

func TestKindKeyResetsTheValueToTheKindDefault(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Value = availGiB
	src := &fakeSource{results: []result{{metrics: samples(1)}, {metrics: cpuSamples(50)}}}
	m := tui.New(cfg, src, tui.WithClock(now))
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})
	m, _ = poll(t, m, m.Init())

	m, fetch := update(t, m, key("k"))
	m, _ = update(t, m, fetch())

	assertContains(t, m, "used_pct  latest 50.0%")
}

func TestValueKeyCyclesSeriesWithoutFetching(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20)}}}
	m, _ := start(t, src, 100, 20)

	m, cmd := update(t, m, key("v"))
	if cmd != nil {
		t.Error("v returned a command")
	}

	assertContains(t, m, "avail_gib  latest 4.00")

	m, _ = update(t, m, key("v"))
	assertContains(t, m, "used_pct  latest 20.0%")

	if n := len(src.Calls()); n != 1 {
		t.Errorf("v made requests: %d calls, want 1", n)
	}
}

func TestWindowKeys(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20, 30)}, {metrics: samples(50)}}}
	m, _ := start(t, src, 100, 20)

	m, cmd := update(t, m, key("-"))
	if cmd != nil {
		t.Error("narrowing started a fetch, the buffer already covers it")
	}

	assertContains(t, m, "window 1m0s", "over 1m0s", "latest 30.0%")

	if m, cmd = update(t, m, key("-")); cmd != nil {
		t.Error("narrowing past the shortest preset started a fetch")
	}

	assertContains(t, m, "window 1m0s")

	m, fetch := update(t, m, key("+"))
	if fetch == nil {
		t.Fatal("widening did not start a fetch")
	}

	assertContains(t, m, "window 5m0s")

	m, _ = update(t, m, fetch())

	if want := (call{model.KindMemory, now().Add(-5 * time.Minute), now()}); lastCall(t, src) != want {
		t.Errorf("widening request = %v, want the whole window %v", lastCall(t, src), want)
	}

	// The result replaced the buffer instead of being merged after it.
	assertContains(t, m, "latest 50.0%", "min 50.0%", "max 50.0%")

	for _, want := range []string{"window 15m0s", "window 1h0m0s"} {
		m, _ = update(t, m, key("="))
		assertContains(t, m, want)
	}

	if m, cmd = update(t, m, key("+")); cmd != nil {
		t.Error("widening past the longest preset started a fetch")
	}

	assertContains(t, m, "window 1h0m0s")
}

// pausedModel returns a model that polled samples(10, 20, 30) once and was paused at now while its second fetch
// was in flight, with that fetch, the poll that started it, and a clock that later calls can move.
//
//nolint:ireturn // The poll is replayed as the tea.Msg its tick produced.
func pausedModel(t *testing.T, src *fakeSource) (tui.Model, tea.Cmd, tea.Msg, *time.Time) {
	t.Helper()

	clock := now()
	m := tui.New(testConfig(), src, tui.WithClock(func() time.Time { return clock }))
	m, _ = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})

	m, tick := update(t, m, m.Init()())

	// A tick command fires once, so keep its message to replay it later.
	oldPoll := tick()
	m, inFlight := update(t, m, oldPoll)

	m, cmd := update(t, m, key("space"))
	if cmd != nil {
		t.Error("pausing returned a command")
	}

	return m, inFlight, oldPoll, &clock
}

func TestPauseStopsPollingAndFreezesTheView(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20, 30)}}}
	m, inFlight, oldPoll, clock := pausedModel(t, src)
	*clock = clock.Add(10 * time.Second)

	assertContains(t, m, "PAUSED  used_pct", "(1s ago)", "space resume")

	// The fetch in flight when pausing still lands, but schedules no poll.
	m, cmd := update(t, m, inFlight())
	if cmd != nil {
		t.Error("a result while paused scheduled a poll")
	}

	if _, cmd = update(t, m, oldPoll); cmd != nil {
		t.Error("a poll while paused started a fetch")
	}

	assertContains(t, m, "(1s ago)")

	// Widening while paused reads the window ending at the pause, once.
	m, fetch := update(t, m, key("+"))
	if _, cmd = update(t, m, fetch()); cmd != nil {
		t.Error("a fetch while paused scheduled a poll")
	}

	if got := lastCall(t, src); !got.to.Equal(now()) || !got.from.Equal(now().Add(-15*time.Minute)) {
		t.Errorf("request while paused = %v, want the 15m window ending at the pause", got)
	}
}

func TestResumeStartsOnePollChain(t *testing.T) {
	t.Parallel()

	src := &fakeSource{results: []result{{metrics: samples(10, 20, 30)}}}
	m, _, oldPoll, clock := pausedModel(t, src)
	*clock = clock.Add(10 * time.Second)

	m, fetch := update(t, m, key("p"))
	if fetch == nil {
		t.Fatal("resuming did not start a fetch")
	}

	m, next := update(t, m, fetch())
	if next == nil {
		t.Error("the first result after resuming did not schedule a poll")
	}

	if got := lastCall(t, src); !got.to.Equal(*clock) || !got.from.Equal(now().Add(-time.Second)) {
		t.Errorf("request after resuming = %v, want from the newest sample to now", got)
	}

	if out := screen(m); strings.Contains(out, "PAUSED") {
		t.Errorf("view still shows PAUSED after resuming:\n%s", out)
	}

	// A poll scheduled before the pause belongs to the old chain.
	if _, cmd := update(t, m, oldPoll); cmd != nil {
		t.Error("a poll from before the pause started a fetch after resuming")
	}
}
