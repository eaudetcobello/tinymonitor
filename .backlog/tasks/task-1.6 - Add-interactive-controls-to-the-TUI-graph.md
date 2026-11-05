---
id: TASK-1.6
title: Add interactive controls to the TUI graph
status: In Progress
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T05:10:00Z'
labels:
  - tui
dependencies:
  - TASK-1.5
references:
  - collector
  - daemon-entrypoint
  - live-graph
  - internal/collector/collector.go
  - cmd/tinymonitor/main.go
  - internal/tui/model.go
  - internal/tui/series.go
modified_files:
  - internal/collector/collector.go
  - cmd/tinymonitor/main.go
  - internal/collector/collector_internal_test.go
  - internal/tui/series.go
  - internal/tui/series_internal_test.go
  - internal/tui/config.go
  - internal/tui/controls.go
  - internal/tui/model.go
  - internal/tui/controls_internal_test.go
  - internal/tui/model_test.go
  - internal/tui/config_test.go
parent_task_id: TASK-1
priority: low
type: enhancement
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
With a working graph, the next friction is restarting the command to look at another series or time span. Keyboard controls let the user explore without leaving the app.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A key cycles through the value series of the current kind (e.g. used_pct, avail_gib)
- [x] #2 Keys widen and narrow the time window between sensible presets (e.g. 1m, 5m, 15m, 1h)
- [x] #3 A key pauses and resumes live refresh
- [x] #4 A help footer lists the available keys
- [x] #5 The daemon collects and stores a cpu metric (used_pct, user_pct, system_pct) alongside memory
- [x] #6 A key switches the graph between the cpu and memory kinds without restarting
- [ ] #7 Downsampling buckets sit on a fixed time grid, so the averages of past buckets stay the same between refreshes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
0. Branch: task-1.6-tui-controls from master (TASK-1.5 is merged).

1. CPU collection in the daemon (scope added by the user: AC #5, #6)
   - collector.Collector gains CPU() (model.Metric, error). Gopsutil becomes a pointer type holding the previous cpu.Times(false) sample, since usage is a delta between two readings.
   - Values (all host-wide, 0-100): used_pct (1 - idle delta / total delta, idle includes iowait as gopsutil does), user_pct, system_pct. Same hard-coded agent ID as Memory.
   - Pure helper cpuUsage(prev, cur cpu.TimesStat) map[string]float64; a zero or negative total delta (clock reset, same reading twice) returns an error so no 0% or NaN sample is stored.
   - The first call only primes the baseline and returns collector.ErrNoBaseline; monitor() skips that error silently.
   - cmd/tinymonitor monitor(): each 500 ms tick collects and stores memory, then CPU. A failure of one kind is logged and does not skip the other. Puts stay synchronous (2 per tick on DynamoDB local is fine).
   - TUI defaultValue(KindCPU) becomes used_pct.
   - Not touched: internal/agent (unused Job scaffolding), the API and store (kind=cpu is already valid end to end).

2. Runtime view state in the TUI model
   - Model gets kind, window and want (the requested value, "" = kind default) initialised from Config. Header, status, placeholders, fetch and rebuild read these instead of cfg.Kind / cfg.Window / cfg.Value. Config is unchanged; flags set the starting view.
   - gen int: bumped by every control that starts a fetch outside the poll chain. pollMsg and fetchedMsg carry the gen they were created under; Update drops messages from an older gen. This keeps exactly one poll chain alive and stops a late answer for the old kind or window landing in the new buffer.
   - fetchedMsg gains replace bool: the result replaces the buffer instead of merging into it (used after widening).
   - fetch() start: max(last buffered time, to - window), so resuming after a long pause or narrowing never reloads more than one window.

3. Keys (AC #1-#4, #6). Proposed bindings, all shown in the footer:
   - k: cycle kind through model.Kinds() (cpu, memory). Clears the buffer, resets want to "" (the new kind's default, so -value avail_gib does not carry over to cpu), loaded=false ("loading..."), bumps gen, fetches the whole window.
   - v: cycle value to the next name in the sorted available list of the current data (wraps; when the current value is not in the list, e.g. a bad -value, goes to the first). Rebuild only, no fetch: the buffer already holds every value (TASK-1.5 design). No-op before data arrives.
   - + / = : widen to the next preset above the current window; - : narrow to the next preset below. Presets 1m, 5m, 15m, 1h. A non-preset -window (e.g. 10m) steps to the nearest preset in that direction; presets below -refresh are skipped (Config requires refresh <= window); at either end the key is a no-op.
     Narrowing: rebuild only (the buffer already covers it; merge trims on the next poll).
     Widening: bump gen, fetch [to - window, to] with replace=true. The old graph stays visible, rescaled to the wider axis, until the answer lands.
   - space (and p): pause/resume. Pause freezes time at pausedAt: rebuild, the status age and any fetch use pausedAt instead of now(). A pollMsg arriving while paused is dropped (chain ends); a fetch already in flight still lands but schedules no tick. Resume bumps gen and fetches from the last buffered sample, starting a single new chain.
     While paused, k and +/- still work against the frozen time (one fetch, no follow-up tick), so the user can explore a frozen moment.
   - q / ctrl+c: quit (unchanged).

4. View (AC #4)
   - Footer (faint): "k kind  v value  +/- window  space pause  q quit". Truncated to the terminal width like the other lines.
   - Header shows the live kind and window. Status line shows "PAUSED" (yellow, bold) at the start while paused.
   - Layout and the exact width x height guarantee are unchanged (footer stays one line).

5. Pure helpers (series.go or new controls.go, white-box tests)
   - nextWindow(cur, refresh, dir) time.Duration over the presets.
   - nextValue(cur string, available []string) string.
   - nextKind(cur model.Kind) model.Kind over model.Kinds().

6. Tests
   - collector_internal_test.go: cpuUsage on known TimesStat pairs (used/user/system math), zero total delta errors; Gopsutil.CPU returns ErrNoBaseline on the first call (live gopsutil, only checks the error kind and then 0..100 ranges on a second call).
   - series/controls tests: table tests for nextWindow (presets, non-preset start, refresh floor, ends), nextValue (wrap, missing current, empty), nextKind, defaultValue(cpu).
   - model_test.go (fake source records kind as well):
     k switches kind: next call asks for cpu over the whole window, view shows "loading..." then cpu data; a fetchedMsg from the previous gen is dropped (no merge, no tick).
     v cycles the status line from used_pct to the next name without calling the source.
     + widens: call covers the new window with replace; - narrows without a call; ends are no-ops; header shows the new window.
     space: a pollMsg while paused returns no cmd; fetchedMsg while paused returns no tick; status shows PAUSED and the age is frozen while the clock advances; resume returns a fetch cmd and a stale pollMsg is dropped.
     Footer lists every key. Existing size test still passes at all four sizes with the longer footer.

7. Verify
   - go build ./..., go vet ./..., go test -race ./..., golangci-lint run ./...
   - Manual (pty) against dynamodb-local + daemon: cpu samples are stored (curl /metrics?kind=cpu), k toggles cpu/memory, v cycles values, +/- step through presets with the graph rescaling and widening backfilled, space freezes the graph and the age, resume catches up, footer visible.

Commits: (1) CPU collector + monitor loop with tests, (2) TUI runtime state + gen + fetch range, (3) keys + footer with tests. Groma: record modified files per the backlog guide; update the collector description (memory and CPU) at the end if it says memory only.

Out of scope: per-core CPU, load average, multiple series on one chart, configurable presets, reverse-cycle keys.

8. Stable buckets (added after review)
   - The bucket edges start at now - window, which moves by refresh + request time on every poll, so samples change buckets and the whole line wobbles.
   - New pure helper gridRange(to, window, buckets) -> (from, to, bucket): bucket = window / buckets, end = to.Truncate(bucket) + bucket (end of the bucket holding now), start = end - bucket x buckets. Truncate is relative to absolute time, so a sample always falls into the same bucket while the width and window stay the same.
   - rebuild: compute buckets from GraphWidth after setting the Y range, then use the grid range for downsample, maxGap and the chart X range. The status line keeps using the true window [now - window, now].
   - A resize, a window change or a wider Y label changes the bucket width and moves the grid once; that is expected.
   - Tests: gridRange alignment and span; downsample on the grid at two view times less than a bucket apart gives identical points for every complete bucket.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Branch: task-1.6-tui-controls from master.

Deviations from the plan:
- Two commits instead of three: the runtime view state and the keys landed together, since the state only exists to serve the keys.
- Key handling is split into handleKey, handleFetched, setWindow and togglePause to stay under the cyclomatic limit; window presets are a function (no globals).
- While paused, the error line reads "error (resume to retry): ..." since no retry happens.
- Groma: the architecture map lives on the groma-curation branch, not master, so the collector description (memory only) is not updated here.

Verification: go vet, go test -race ./..., golangci-lint run ./... (0 issues). Live: GET /metrics?kind=cpu returns used_pct/user_pct/system_pct samples at 2/s from the daemon. TUI driven in a pty (pyte) against the running daemon: v cycles avail_gib -> total_gib, k switches memory -> cpu -> memory with the header updating, v on cpu shows user_pct, - goes to 1m, + to 5m then 15m, space shows PAUSED with the age frozen for 3s and the footer reading 'space resume', p resumes, q exits 0.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
The TUI graph is now explorable without restarting, and the daemon records CPU usage.

- Daemon: collector.Gopsutil gains CPU(), computing used_pct, user_pct and system_pct from the change between two cpu.Times readings (first call primes the baseline). monitor() stores a memory and a CPU sample each tick; one failing does not skip the other.
- TUI keys: k cycles cpu/memory (reloads the window, resets to the kind's default value), v cycles the value series (no request), +/- steps through 1m, 5m, 15m, 1h (widening reloads the whole window, narrowing only redraws), space/p pauses and resumes (pause freezes the axis and the age; keys still work on the frozen moment). The footer lists the keys and the status line shows PAUSED.
- Keys that start a fetch bump a generation counter; late polls and results from an older generation are dropped, so one poll chain runs and an old kind's answer never lands in the new buffer. Polls ask for at most one window, so resuming after a long pause stays cheap.
- Tests: CPU math and baseline, preset/value/kind stepping, and model tests for each key including stale results, replace on widen, and pause/resume chains.

Follow-up: the Groma map on groma-curation still describes the collector as memory only.
<!-- SECTION:FINAL_SUMMARY:END -->
