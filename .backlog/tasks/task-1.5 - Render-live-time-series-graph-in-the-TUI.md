---
id: TASK-1.5
title: Render live time-series graph in the TUI
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - tui
dependencies:
  - TASK-1.3
  - TASK-1.4
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
- [ ] #1 The TUI polls the API at the refresh interval and draws a line graph of the selected value over the configured rolling time window
- [ ] #2 The graph shows labeled Y-axis bounds, the time span on the X axis, and the latest value
- [ ] #3 Polling runs off the UI loop so a slow or failing request never freezes input
- [ ] #4 When the server is unreachable or returns an error, the UI shows the error and keeps retrying; the last good data stays on screen
- [ ] #5 An empty result shows a "no data" state instead of a blank or broken graph
- [ ] #6 Data-to-series transformation (windowing, downsampling to terminal width) is covered by unit tests
<!-- AC:END -->
