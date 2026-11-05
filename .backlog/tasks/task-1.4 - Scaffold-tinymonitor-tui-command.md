---
id: TASK-1.4
title: Scaffold tinymonitor-tui command
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T04:34:00Z'
labels:
  - tui
dependencies:
  - TASK-1.1
references:
  - tinymonitor
  - client
  - metric-model
modified_files:
  - internal/tui/config.go
  - internal/tui/config_test.go
  - go.mod
  - go.sum
  - internal/tui/model.go
  - internal/tui/model_test.go
  - cmd/tinymonitor-tui/main.go
parent_task_id: TASK-1
priority: medium
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The TUI needs its own binary with configuration and a terminal app lifecycle before any graph exists. Pick and record the TUI and charting libraries in this task (Bubble Tea with a chart component such as ntcharts is a likely candidate; verify current docs and maintenance status first).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `go run ./cmd/tinymonitor-tui` opens a full-screen terminal app and `q` / Ctrl+C exit cleanly, restoring the terminal
- [x] #2 Flags: server URL (default `http://localhost:1323`), agent ID, metric kind, time window, refresh interval; `--help` documents each
- [x] #3 Invalid flag values exit non-zero with a clear message before the UI starts
- [x] #4 The layout adapts on terminal resize
- [x] #5 The library choice and reason are recorded in the task notes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
0. Branch: create task-1.4-tui-scaffold from task-1.3-metrics-client (task branches stay stacked, master is still at the first commit).

1. Libraries (record in notes, AC #5)
   - charm.land/bubbletea/v2 v2.1.1 (released 2026-10-28). v2 is the maintained line; v1 (github.com/charmbracelet/bubbletea) last released v1.3.10 in 2025-09. Model-View-Update fits a polling graph: fetches run as tea.Cmd off the UI loop, which TASK-1.5 AC #3 needs.
   - github.com/NimbleMarkets/ntcharts/v2 v2.9.0 (released 2026-10-29) for charts. It is built on bubbletea/v2, lipgloss/v2 and bubbles/v2, and ships timeserieslinechart, which matches TASK-1.5 (time on X, value on Y). The v0.5.x line is still on Bubble Tea v1, so v2 is the only compatible choice.
   - charm.land/lipgloss/v2 v2.0.6 for layout (same versions ntcharts/v2 pins, no version skew).
   - This task only adds bubbletea/v2 and lipgloss/v2 to go.mod. ntcharts/v2 is added by TASK-1.5 when the graph exists, since go mod tidy would drop an unused import.

2. Config (new internal/tui/config.go)
   - type Config {ServerURL string; AgentID uint32; Kind model.Kind; Window, Refresh time.Duration}.
   - ParseConfig(args []string, output io.Writer) (Config, error) on a flag.NewFlagSet with ContinueOnError, so it is testable and main decides exit codes.
   - Flags and defaults:
     --server   http://localhost:1323   tinymonitor API base URL
     --agent    123123123               agent ID (the ID the collector currently hardcodes, so the default shows data)
     --kind     memory                  metric kind, one of cpu, memory
     --window   5m                      rolling time window shown on the graph
     --refresh  1s                      poll interval
   - Validation, each with a clear message naming the flag and value:
     server: validated by calling client.New (reuses ErrInvalidBaseURL, no duplicated rules); main keeps the returned *client.Client for TASK-1.5.
     agent: parsed through flag.Func with strconv.ParseUint(s, 10, 32), so negatives and values over uint32 are rejected.
     kind: model.ParseKind.
     window and refresh: must be > 0; refresh must not exceed window.
   - Custom flag.Usage prints a one-line description, usage line and PrintDefaults, so --help documents each flag (AC #2).

3. Entry point (new cmd/tinymonitor-tui/main.go, kept thin like cmd/tinymonitor)
   - ParseConfig(os.Args[1:], os.Stderr): flag.ErrHelp exits 0; any other error prints "tinymonitor-tui: <err>" to stderr and exits 2 before the UI starts (AC #3).
   - signal.NotifyContext(SIGTERM) and tea.NewProgram(tui.New(cfg), tea.WithContext(ctx)).Run(). A run ending in context cancellation (ErrProgramKilled wrapping context.Canceled) exits 0; other errors print and exit 1. Bubble Tea restores the terminal in both paths.
   - Ctrl+C arrives as a key press in raw mode, so the model handles it (step 4) rather than a signal.

4. Model (new internal/tui/model.go)
   - Model{cfg Config; width, height int}. Init returns nil (polling is TASK-1.5).
   - Update: tea.WindowSizeMsg stores width and height (AC #4). tea.KeyPressMsg with String() "q" or "ctrl+c" returns tea.Quit (AC #1).
   - View returns tea.View with AltScreen = true (v2 sets the alt screen per view, not as a program option) (AC #1).
   - Layout with lipgloss, sized from width and height: a one-line header (server, agent, kind, window, refresh), a body that fills the remaining height with a centered placeholder ("graph: TASK-1.5"; this body is what TASK-1.5 replaces with the chart), and a one-line footer "q quit". Before the first WindowSizeMsg, View returns a short "loading" line. Very small terminals truncate instead of wrapping.

5. Tests (internal/tui, package tui_test, t.Parallel)
   - config_test.go: defaults with no args; each flag overrides; table of invalid values (bad URL, no scheme, agent -1, agent 4294967296, agent abc, unknown kind, window 0, refresh -1s, refresh > window, unknown flag) returns an error mentioning the flag; --help returns flag.ErrHelp and the usage text names every flag.
   - model_test.go: WindowSizeMsg then View renders exactly width x height cells (lipgloss.Width/Height) at two sizes, and AltScreen is set; "q" and ctrl+c return a cmd producing tea.QuitMsg; another key does not quit.

6. Verify
   - go build ./..., go vet ./..., go test -race ./..., golangci-lint run ./...
   - Manual: go run ./cmd/tinymonitor-tui opens full screen, resize reflows, q and Ctrl+C restore the shell; go run ./cmd/tinymonitor-tui --kind disk prints an error and exits 2; --help lists all flags.

Commits: (1) config parsing with tests, (2) Bubble Tea model and cmd/tinymonitor-tui entry point with tests. Groma: link files per the backlog guide; the new tui container is left for a later curation pass.

Out of scope: polling, the chart, error display (TASK-1.5); key controls beyond quit (TASK-1.6); a VS Code launch config for the TUI.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Library choice (AC #5):
- charm.land/bubbletea/v2 v2.1.1 (2026-10-28). v2 is the maintained line; v1 last released v1.3.10 in 2025-09. Its Model-View-Update loop runs fetches as commands off the UI loop, which TASK-1.5 AC #3 needs.
- github.com/NimbleMarkets/ntcharts/v2 v2.9.0 (2026-10-29) for the graph. It is built on bubbletea/v2, lipgloss/v2 and bubbles/v2 and ships timeserieslinechart. ntcharts v0.5.x is still on Bubble Tea v1, so v2 is the only compatible line. Not in go.mod yet: TASK-1.5 adds it with the first import.
- charm.land/lipgloss/v2 v2.0.6 for layout, the version ntcharts/v2 pins.

Deviations from the plan:
- No signal.NotifyContext in main. Bubble Tea v2 already turns SIGTERM into a clean quit and a non-TTY SIGINT into ErrInterrupted; main treats both as exit 0.
- ParseConfig prints nothing on errors (the flag package would print the error and the usage, then main the error again). main prints one line plus a -help hint and exits 2. Only --help writes the usage.
- The agent flag error omits the strconv detail, keeping the message to the accepted range.

Verification: go build, go vet, go test -race ./... and golangci-lint run ./... (0 issues). Binary checks: --help exits 0 and lists all five flags; bad kind, agent -1 / 4294967296 / abc, server without scheme, refresh 0s, refresh > window and a positional arg each exit 2 with one message. In a pty (script and a Python pty driver): the app enters the alt screen, q and Ctrl+C both leave it, restore the cursor and exit 0; a resize from 80x24 to 40x10 redraws with the header truncated to 40 cols, the placeholder centered and the footer on row 10.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Adds the tinymonitor-tui binary: a full-screen Bubble Tea v2 app with validated flags and a layout that TASK-1.5 fills with the graph.

- internal/tui.ParseConfig parses -server (default http://localhost:1323, checked with client.New), -agent (uint32, default 123123123, the ID the collector writes), -kind (model.ParseKind), -window (default 5m) and -refresh (default 1s, > 0 and at most -window). Errors wrap ErrInvalidFlag and name the flag; --help returns flag.ErrHelp after printing the usage.
- internal/tui.Model tracks the terminal size, quits on q and Ctrl+C, and renders exactly width x height cells in the alt screen: a settings header, a centered placeholder body and a 'q quit' footer.
- cmd/tinymonitor-tui exits 2 on invalid flags before the UI starts, 0 on a requested stop and 1 on an app error.

Libraries: Bubble Tea v2 and Lip Gloss v2 now; ntcharts/v2 is chosen for TASK-1.5 (reasons in the notes).

Tests cover defaults, overrides, 13 invalid inputs, --help content, view size at four terminal sizes, and the quit keys. Lint, race tests and pty runs pass (details in the notes).

Follow-up: none. The new tui container is not in the Groma architecture yet; a later curation pass adds it.
<!-- SECTION:FINAL_SUMMARY:END -->
