---
id: TASK-1.6
title: Add interactive controls to the TUI graph
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - tui
dependencies:
  - TASK-1.5
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
- [ ] #1 A key cycles through the value series of the current kind (e.g. used_pct, avail_gib)
- [ ] #2 Keys widen and narrow the time window between sensible presets (e.g. 1m, 5m, 15m, 1h)
- [ ] #3 A key pauses and resumes live refresh
- [ ] #4 A help footer lists the available keys
<!-- AC:END -->
