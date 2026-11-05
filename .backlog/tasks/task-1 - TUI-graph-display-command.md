---
id: TASK-1
title: TUI graph display command
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - tui
dependencies: []
priority: medium
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
tinymonitor collects host metrics into DynamoDB and exposes them over `GET /metrics`, but there is no way to look at them besides raw HTTP calls. A separate terminal command that graphs metrics live gives a quick view of an agent without a web UI. This parent groups the work: entrypoint split, a working read path, and the TUI itself. The TUI talks to the HTTP API rather than DynamoDB directly so it can run on another machine than the server.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A second binary under `cmd/` renders live metric graphs in the terminal from the tinymonitor HTTP API
- [ ] #2 All subtasks are Done
<!-- AC:END -->
