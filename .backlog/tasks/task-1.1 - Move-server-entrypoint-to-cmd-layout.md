---
id: TASK-1.1
title: Move server entrypoint to cmd/ layout
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - tui
dependencies: []
parent_task_id: TASK-1
priority: high
type: chore
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The repo has a single `main.go` at the root that starts the collector loop and HTTP server. Adding a second binary (the TUI) needs the standard Go `cmd/<name>/main.go` layout so each program builds and runs independently.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The existing server/collector program lives at `cmd/tinymonitor/main.go` and behaves as before
- [ ] #2 No `main` package remains at the repo root
- [ ] #3 `go build ./...` and `golangci-lint run` pass
- [ ] #4 `.vscode/launch.json` points at the new entrypoint
<!-- AC:END -->
