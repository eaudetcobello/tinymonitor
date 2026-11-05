---
id: TASK-1
title: TUI graph display command
status: Done
assignee: []
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T05:13:00Z'
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
- [x] #1 A second binary under `cmd/` renders live metric graphs in the terminal from the tinymonitor HTTP API
- [x] #2 All subtasks are Done
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
tinymonitor now has a terminal view of its metrics: cmd/tinymonitor-tui polls GET /metrics through internal/client (so it can run on another machine than the daemon) and draws a live braille graph of one value over a rolling window.

- TASK-1.1 moved the daemon to cmd/tinymonitor; TASK-1.2 implemented the read path behind GET /metrics; TASK-1.3 added the Go HTTP client.
- TASK-1.4 scaffolded the TUI command and its flags; TASK-1.5 rendered the live graph with gap handling, error recovery and placeholders.
- TASK-1.6 added keys for kind (cpu/memory), value, window presets and pause, CPU collection in the daemon, and grid-aligned downsampling so the line holds steady.

Verified in each subtask with go test -race, golangci-lint and live runs against DynamoDB local and the daemon. TASK-1.6 is under review in PR #8.

Follow-up: the Groma map on groma-curation still describes the collector as memory only.
<!-- SECTION:FINAL_SUMMARY:END -->
