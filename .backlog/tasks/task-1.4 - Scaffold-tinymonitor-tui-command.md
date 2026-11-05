---
id: TASK-1.4
title: Scaffold tinymonitor-tui command
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - tui
dependencies:
  - TASK-1.1
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
- [ ] #1 `go run ./cmd/tinymonitor-tui` opens a full-screen terminal app and `q` / Ctrl+C exit cleanly, restoring the terminal
- [ ] #2 Flags: server URL (default `http://localhost:1323`), agent ID, metric kind, time window, refresh interval; `--help` documents each
- [ ] #3 Invalid flag values exit non-zero with a clear message before the UI starts
- [ ] #4 The layout adapts on terminal resize
- [ ] #5 The library choice and reason are recorded in the task notes
<!-- AC:END -->
