package tui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"github.com/eaudetcobello/tinymonitor/internal/client"
	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const (
	// minRequestTimeout is the shortest request timeout. Requests time out after 2 × -refresh,
	// clamped to [minRequestTimeout, client.DefaultTimeout].
	minRequestTimeout = 2 * time.Second
	// minChartWidth and minChartHeight are the smallest body that still fits a readable chart.
	minChartWidth  = 20
	minChartHeight = 3
	// xLabelStep is the spacing in columns between X axis time labels.
	xLabelStep = 12
	// yLabelStep is the spacing in rows between Y axis value labels.
	yLabelStep = 2
	// brailleDotsPerCell is the horizontal resolution of a braille cell.
	brailleDotsPerCell = 2
)

// MetricsSource reads the metrics of one kind for an agent within [from, to], ordered by time ascending.
// *client.Client implements it.
type MetricsSource interface {
	Get(ctx context.Context, agentID uint32, kind model.Kind, from, to time.Time) ([]model.Metric, error)
}

var _ MetricsSource = (*client.Client)(nil)

// Option customizes a Model.
type Option func(*Model)

// WithClock replaces time.Now as the source of the current time.
func WithClock(now func() time.Time) Option {
	return func(m *Model) { m.now = now }
}

// pollMsg asks the Model to start the next fetch.
type pollMsg struct{}

// fetchedMsg carries the result of one fetch that ended its range at to.
type fetchedMsg struct {
	metrics []model.Metric
	err     error
	to      time.Time
}

// Model is the Bubble Tea model of tinymonitor-tui. It polls src every cfg.Refresh and draws the selected value
// over the rolling cfg.Window, under a header with the settings and above a footer with the keys.
type Model struct {
	cfg Config
	src MetricsSource
	now func() time.Time

	width, height int

	// metrics holds the samples of the window, ordered by time.
	metrics []model.Metric
	// loaded is true once a fetch has succeeded.
	loaded bool
	// err is the error of the last fetch, nil when it succeeded.
	err error

	// The fields below are derived from metrics by rebuild.
	value     string
	available []string
	points    []point
	chart     string
}

// New returns a Model that graphs the metrics src returns for cfg.
func New(cfg Config, src MetricsSource, opts ...Option) Model {
	m := Model{cfg: cfg, src: src, now: time.Now}
	for _, opt := range opts {
		opt(&m)
	}

	return m.rebuild()
}

// Init implements tea.Model: it starts the first fetch.
func (m Model) Init() tea.Cmd {
	return m.fetch()
}

// Update implements tea.Model. Fetches run as commands, outside the UI loop; each result schedules the next poll
// one refresh later, so requests never overlap.
//
//nolint:ireturn // tea.Model requires returning the interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m = m.rebuild()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case pollMsg:
		return m, m.fetch()
	case fetchedMsg:
		m.err = msg.err
		if msg.err == nil {
			m.metrics = merge(m.metrics, msg.metrics, msg.to.Add(-m.cfg.Window))
			m.loaded = true
		}

		m = m.rebuild()

		return m, tea.Tick(m.cfg.Refresh, func(time.Time) tea.Msg { return pollMsg{} })
	}

	return m, nil
}

// View implements tea.Model. It renders full screen, exactly the size of the terminal.
func (m Model) View() tea.View {
	v := tea.NewView("loading…")
	v.AltScreen = true

	if m.width <= 0 || m.height <= 0 {
		return v
	}

	line := lipgloss.NewStyle().Inline(true).MaxWidth(m.width)
	header := line.Render(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf(
		"tinymonitor  %s  agent %d  %s  window %s  refresh %s",
		m.cfg.ServerURL, m.cfg.AgentID, m.cfg.Kind, m.cfg.Window, m.cfg.Refresh,
	)))
	rows := []string{header, line.Render(m.status())}

	if m.err != nil {
		rows = append(rows, line.Render(lipgloss.NewStyle().Foreground(lipgloss.Red).Render(fmt.Sprintf(
			"error (retrying every %s): %s", m.cfg.Refresh, errorText(m.err)))))
	}

	w, h := m.bodySize()

	body := m.chart
	if body == "" {
		body = lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, line.Render(m.placeholder()))
	}

	rows = append(rows, body, line.Render(lipgloss.NewStyle().Faint(true).Render("q quit")))

	v.SetContent(lipgloss.NewStyle().
		Width(m.width).MaxWidth(m.width).
		Height(m.height).MaxHeight(m.height).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...)))

	return v
}

// fetch returns a command that reads the samples the buffer is missing: the whole window when it is empty,
// otherwise everything from the newest buffered sample on.
func (m Model) fetch() tea.Cmd {
	to := m.now()

	from := to.Add(-m.cfg.Window)
	if n := len(m.metrics); n > 0 {
		from = m.metrics[n-1].Time
	}

	src, cfg := m.src, m.cfg
	timeout := min(max(2*cfg.Refresh, minRequestTimeout), client.DefaultTimeout)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		metrics, err := src.Get(ctx, cfg.AgentID, cfg.Kind, from, to)

		return fetchedMsg{metrics: metrics, err: err, to: to}
	}
}

// bodySize returns the size of the area between the status lines and the footer.
func (m Model) bodySize() (int, int) {
	lines := 3 // header, status, footer
	if m.err != nil {
		lines++
	}

	return m.width, max(m.height-lines, 0)
}

// rebuild returns m with the series and the chart recomputed from the buffered metrics and the terminal size.
func (m Model) rebuild() Model {
	to := m.now()
	from := to.Add(-m.cfg.Window)

	m.value, m.available = selectValue(m.cfg.Value, m.cfg.Kind, m.metrics)
	m.points = seriesFor(m.metrics, m.value, from, to)
	m.chart = ""

	w, h := m.bodySize()
	if len(m.points) == 0 || w < minChartWidth || h < minChartHeight {
		return m
	}

	lo, hi, _ := bounds(m.points)
	value := m.value
	chart := timeserieslinechart.New(w, h,
		timeserieslinechart.WithXYSteps(xLabelStep, yLabelStep),
		timeserieslinechart.WithXLabelFormatter(func(_ int, v float64) string {
			return time.UnixMilli(int64(v * 1e3)).Format("15:04:05")
		}),
		timeserieslinechart.WithYLabelFormatter(func(_ int, v float64) string {
			return formatNumber(value, v)
		}),
	)
	// Keep the axes on the window and the bounds; the chart would otherwise widen them to fit each pushed point.
	chart.AutoMinX, chart.AutoMaxX, chart.AutoMinY, chart.AutoMaxY = false, false, false, false
	chart.SetTimeRange(from, to)
	chart.SetViewTimeRange(from, to)
	chart.SetYRange(lo, hi)
	chart.SetViewYRange(lo, hi)

	buckets := chart.GraphWidth() * brailleDotsPerCell
	gap := maxGap(m.points, m.cfg.Window/time.Duration(max(buckets, 1)))

	// Each run is its own data set, so the chart leaves a gap between runs instead of joining them.
	var names []string

	for i, run := range splitRuns(m.points, gap) {
		name := fmt.Sprintf("run%d", i)
		names = append(names, name)

		chart.SetDataSetStyle(name, lipgloss.NewStyle().Foreground(lipgloss.Cyan))

		for _, p := range downsample(run, from, to, buckets) {
			chart.PushDataSet(name, timeserieslinechart.TimePoint{Time: p.t, Value: p.v})
		}
	}

	chart.DrawBrailleDataSets(names)
	m.chart = chart.View()

	return m
}

// status returns the line above the chart: the series, its latest value and its range over the window.
func (m Model) status() string {
	if len(m.points) == 0 {
		return m.value
	}

	latest := m.points[len(m.points)-1]
	lo, hi := latest.v, latest.v

	for _, p := range m.points {
		lo, hi = min(lo, p.v), max(hi, p.v)
	}

	return fmt.Sprintf("%s  latest %s (%s ago)  min %s  max %s  over %s",
		lipgloss.NewStyle().Bold(true).Render(m.value),
		lipgloss.NewStyle().Bold(true).Render(formatValue(m.value, latest.v)),
		m.now().Sub(latest.t).Truncate(time.Second),
		formatValue(m.value, lo), formatValue(m.value, hi), m.cfg.Window)
}

// placeholder returns the text shown in place of the chart.
func (m Model) placeholder() string {
	w, h := m.bodySize()

	switch {
	case !m.loaded && m.err != nil:
		return "no data yet, retrying"
	case !m.loaded:
		return "loading…"
	case len(m.metrics) == 0:
		return fmt.Sprintf("no data in the last %s for agent %d (%s)", m.cfg.Window, m.cfg.AgentID, m.cfg.Kind)
	case len(m.points) == 0:
		return fmt.Sprintf("no %q in the last %s (available: %s)",
			m.value, m.cfg.Window, strings.Join(m.available, ", "))
	case w < minChartWidth || h < minChartHeight:
		return "terminal too small for the graph"
	default:
		return ""
	}
}

// formatNumber formats v of the named value for an axis label: one decimal for percentages, two otherwise.
func formatNumber(name string, v float64) string {
	if strings.HasSuffix(name, "_pct") {
		return fmt.Sprintf("%.1f", v)
	}

	return fmt.Sprintf("%.2f", v)
}

// formatValue formats v of the named value with its unit.
func formatValue(name string, v float64) string {
	if strings.HasSuffix(name, "_pct") {
		return formatNumber(name, v) + "%"
	}

	return formatNumber(name, v)
}

// errorText returns err on one line. For a failed HTTP request it drops the request URL, which the header shows.
func errorText(err error) string {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		err = urlErr.Err
	}

	return strings.Join(strings.Fields(err.Error()), " ")
}
