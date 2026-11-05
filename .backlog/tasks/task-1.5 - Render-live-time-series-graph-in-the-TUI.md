---
id: TASK-1.5
title: Render live time-series graph in the TUI
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T04:50:00Z'
labels:
  - tui
dependencies:
  - TASK-1.3
  - TASK-1.4
references:
  - internal/tui/model.go
  - internal/tui/config.go
  - cmd/tinymonitor-tui/main.go
  - live-graph
  - metrics-client
modified_files:
  - internal/tui/series.go
  - internal/tui/series_internal_test.go
  - internal/tui/config.go
  - internal/tui/config_test.go
  - internal/tui/model.go
  - cmd/tinymonitor-tui/main.go
  - go.mod
  - go.sum
  - internal/tui/model_test.go
parent_task_id: TASK-1
priority: medium
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
This is the core of the feature: seeing how a metric moves over time from the terminal. The collector writes a memory sample every 500ms with values `used_pct`, `avail_gib` and `total_gib`; `used_pct` is the most useful default series.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The TUI polls the API at the refresh interval and draws a line graph of the selected value over the configured rolling time window
- [x] #2 The graph shows labeled Y-axis bounds, the time span on the X axis, and the latest value
- [x] #3 Polling runs off the UI loop so a slow or failing request never freezes input
- [x] #4 When the server is unreachable or returns an error, the UI shows the error and keeps retrying; the last good data stays on screen
- [x] #5 An empty result shows a "no data" state instead of a blank or broken graph
- [x] #6 Data-to-series transformation (windowing, downsampling to terminal width) is covered by unit tests
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
0. Branch: create task-1.5-tui-graph from task-1.4-tui-scaffold (task branches stay stacked).

1. Dependency
   - Add github.com/NimbleMarkets/ntcharts/v2 v2.9.0 (chosen in TASK-1.4). It pins bubbletea/v2 v2.1.1 and lipgloss/v2 v2.0.6, the versions already in go.mod, so no version skew. Brings bubbles/v2 and bubblezone/v2 as indirect deps.
   - Use linechart/timeserieslinechart: New(w, h), Resize, ClearAllData, Push(TimePoint), SetTimeRange/SetViewTimeRange, SetYRange/SetViewYRange, DrawBraille, View. It draws the axes with tick labels; time labels use its HH:MM:SS formatter.

2. Selected value (needs approval: new flag)
   - New flag -value (default empty). Empty means the kind's default series: used_pct for memory; for a kind without a known default, the first value name (sorted) present in the data.
   - Config gains Value string. No validation at parse time beyond trimming: value names come from the data. TASK-1.6 cycles this field at runtime.
   - If the data has samples but none carry the selected value, the body shows: no "<value>" in the last <window> (available: a, b).

3. Data layer (new internal/tui/series.go, pure functions, no Bubble Tea)
   - buffer: []model.Metric for the configured kind, ordered by time. Raw metrics are kept (not one series) so TASK-1.6 can switch values without refetching.
   - merge(buf, fresh, from) []model.Metric: appends samples strictly newer than the last buffered time (the API range is inclusive, so the boundary sample comes back twice), then drops samples older than from.
   - seriesFor(metrics, value, from, to) []TimePoint: samples in [from, to] that carry value, skipping NaN/Inf.
   - downsample(points, from, to, buckets) []TimePoint: splits [from, to] into buckets equal-width time buckets and emits one point per non-empty bucket (mean value, mean time). Empty buckets stay empty, so a collector outage shows as a gap rather than an invented line. Input with <= buckets points is returned unchanged. buckets is the plot width in braille dots (2 per cell).
   - bounds(points) (lo, hi): min and max with 5% padding; a flat series gets a range of +/-1 so the line is visible.

4. Polling (AC #1, #3, #4)
   - tui.New(cfg, src) where src is a MetricsSource interface { Get(ctx, agentID, kind, from, to) ([]model.Metric, error) }. *client.Client satisfies it; tests pass a fake. main builds the client with client.New(cfg.ServerURL, timeout), timeout = clamp(2*refresh, 2s, client.DefaultTimeout).
   - Init returns fetchCmd. fetchCmd is a tea.Cmd, so Bubble Tea runs the HTTP call in its own goroutine and Update never blocks on the network (AC #3). It sends fetchedMsg{metrics, err, at}.
   - Range: first fetch (and after an empty buffer) asks for [now-window, now]. Later fetches ask for [last buffered time, now], so a 1h window at 2 samples/s does not reload 7200 samples every second.
   - On fetchedMsg, Update merges or records the error, then returns tea.Tick(refresh) -> pollMsg -> fetchCmd. The next request starts only after the previous one ends, so a slow server never piles up requests; the cadence is refresh plus request time.
   - On error: keep the buffer and the chart, store the error and its time, keep polling. The next success clears it (AC #4).
   - now is an injectable clock on the Model (unexported field, set by New to time.Now; tests use an option or the internal test package).

5. View (AC #2, #5)
   - Header (unchanged line) + status line + chart body + footer.
   - Status line: <value>  latest <v> (<age> ago)  min <lo>  max <hi>  window <5m>. Latest comes from the newest raw sample, not the downsampled series. Values use %.2f, or %.1f for _pct names with a % suffix.
   - Error line (red, only when the last poll failed): error: <message> - retrying every <refresh>. It takes one row from the chart body.
   - Chart: timeserieslinechart sized to the body, X view range [now-window, now] (time span on the X axis, end labels on the axis), Y range from bounds(), so the Y axis shows labeled min and max. Rebuilt in Update when data or size changes (not in View), redrawn with DrawBraille.
   - States in the body, centered: before the first response "loading..."; first request failed and no data: the error and "retrying"; no samples in the window: no data in the last <window> for agent <id> (<kind>) (AC #5); value missing: see step 2.
   - Terminals too small for a chart (body < 3 rows or < 20 cols) show the status line only. The view keeps rendering exactly width x height cells.

6. Tests
   - series_test.go (package tui, white-box, table-driven, AC #6): merge drops the duplicate boundary sample and samples older than from, keeps order; seriesFor skips other values, NaN and out-of-range times, includes both ends; downsample: fewer points than buckets unchanged, exact means for a known input, empty buckets produce no point, bucket count never exceeds buckets, buckets <= 0 returns nil; bounds: padding, flat series, single point.
   - model_test.go additions (fake source, fixed clock):
     Init returns a cmd that calls the source with [now-window, now] and yields fetchedMsg.
     Update with a pollMsg returns at once with a cmd and does not call the source (the source blocks on a channel; the test fails on timeout if Update calls it) (AC #3).
     Success then error: the view still shows the latest value and contains the error text; the returned cmd is a tick (AC #4).
     Second fetch requests from the last buffered time.
     Empty result shows the no-data text (AC #5); missing value shows the available names.
     Existing size test extended: with data loaded, the view is still exactly width x height at all four sizes.

7. Verify
   - go build ./..., go vet ./..., go test -race ./..., golangci-lint run ./...
   - Manual against the local server (DynamoDB local + collector): graph appears and scrolls, latest value matches the newest stored sample, stopping the server shows the error with the last graph still drawn, restarting it recovers without restarting the TUI, an agent with no data shows the no-data state, resize redraws.

Commits: (1) ntcharts dependency + series.go with tests, (2) -value flag, (3) polling and chart in the model with tests, (4) main wiring. Groma: record modified files per the backlog guide.

Out of scope: key controls for value, window and pause (TASK-1.6); multiple series on one chart; a CPU collector.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Branch: task-1.5-tui-graph from master (TASK-1.4 was merged, so no stacking).

Deviations from the plan:
- Gaps: ntcharts' braille drawing joins every consecutive point, so empty buckets alone cannot show a gap. Samples are split into runs where two neighbours are more than max(4 x median spacing, 2 buckets) apart (splitRuns, maxGap), and each run is its own chart data set.
- downsample always buckets instead of returning small inputs unchanged; with one point per bucket the result is the same.
- The chart's auto range is turned off (AutoMinX/MaxX/MinY/MaxY) so the axes stay on the window and bounds().
- Request timeout lives in the model (context deadline of clamp(2 x refresh, 2s, 10s)); main keeps the client default.
- The error line reads "error (retrying every <refresh>): <cause>" and drops the request URL of *url.Error, so truncation on narrow terminals cuts the cause, not the retry note.
- Main wiring went into the model commit since the New signature change has to build together.
- go get of ntcharts/v2 v2.9.0 raised a few indirect deps (ultraviolet, go-runewidth, xo/terminfo, x/sync, x/sys) to the versions it requires.
- The right-end X label (now) is often dropped by ntcharts when it overlaps the previous label; the status line states the span ("over 5m0s").

Verification: go vet, go test -race ./..., golangci-lint run ./... (0 issues). Live run in a pty (pyte) against dynamodb-local and the daemon: graph scrolls with Y labels at the padded bounds and HH:MM:SS labels on X; killing the daemon shows "error (retrying every 1s): dial tcp [::1]:1323: connect: connection refused" with the last graph still drawn; restarting it recovers without restarting the TUI; -agent 1 shows the no-data text; -value swap_pct lists the available values; q exits 0.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
tinymonitor-tui now draws a live braille line graph of one metric value over the rolling window.

- internal/tui/series.go: pure helpers for merging polled samples into the window buffer, picking the value, extracting a series, splitting it at data gaps, downsampling to the chart width (mean per time bucket) and padded Y bounds. Unit tested (AC #6).
- internal/tui/model.go: polls through a MetricsSource (*client.Client) as Bubble Tea commands; the next poll is scheduled one refresh after the previous result, so requests never overlap or block input. The first fetch reads the whole window, later ones only newer samples. Errors keep the last data on screen and show an error line until the next success. Placeholders for loading, no data, and a value missing from the data.
- New -value flag (default used_pct for memory, else the first value in the data); TASK-1.6 can cycle it at runtime.
- Adds github.com/NimbleMarkets/ntcharts/v2 v2.9.0 for the chart.

Verified with race tests, lint and a live pty run against dynamodb-local (details in the notes). Known limit: the rightmost X label is sometimes hidden by ntcharts' label spacing; the status line states the span.
<!-- SECTION:FINAL_SUMMARY:END -->
