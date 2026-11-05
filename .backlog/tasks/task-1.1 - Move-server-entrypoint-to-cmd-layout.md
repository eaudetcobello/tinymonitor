---
id: TASK-1.1
title: Move server entrypoint to cmd/ layout
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T04:11:00Z'
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
- [x] #1 The existing server/collector program lives at `cmd/tinymonitor/main.go` and behaves as before
- [x] #2 No `main` package remains at the repo root
- [x] #3 `go build ./...` and `golangci-lint run` pass
- [x] #4 `.vscode/launch.json` points at the new entrypoint
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. `git mv main.go cmd/tinymonitor/main.go`. Content unchanged: it stays `package main`, imports are module-absolute (`github.com/eaudetcobello/tinymonitor/internal/...`), and there are no relative file paths or `//go:embed` directives that depend on the working directory.
2. Update `.vscode/launch.json`: `program` -> `${workspaceFolder}/cmd/tinymonitor`, rename the config from "Debug main.go" to "Debug tinymonitor".
3. Confirm nothing else references the root entrypoint (no Makefile, Dockerfile, README or CI config exists today; only Backlog task text mentions `main.go`).
4. Verify: `go build ./...`, `go vet ./...`, `golangci-lint run` (config uses `default: all`, file move should not introduce findings), `ls *.go` at root is empty, and `go build -o /tmp/... ./cmd/tinymonitor` produces a `tinymonitor` binary (same name as before since it matches the module basename).
5. Smoke test behaviour: `go run ./cmd/tinymonitor` against local DynamoDB (localhost:8000) logs `found tables` and `put metric`, then Ctrl+C logs `Stopping...`.
6. Commits (separate, on branch task-1.1-cmd-layout): (a) move main.go to cmd/tinymonitor, (b) launch.json update.

7. (Added on review) Fix the 4 pre-existing wrapcheck errors in internal/server/server.go by adding a wrapcheck ignore-sig-regexps entry for echo v5 Context response writers in .golangci.yml.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Moved main.go to cmd/tinymonitor/main.go unchanged, updated launch.json. go build ./... and go vet ./... pass; ./cmd/tinymonitor builds a tinymonitor binary; no .go files left at root. golangci-lint reports 4 wrapcheck errors in internal/server/server.go (lines 53, 58, 63, 76); same 4 on master, so they predate this task. Runtime smoke test not run: local DynamoDB on :8000 was down.

Lint fix: added linters.settings.wrapcheck.ignore-sig-regexps for (*echo/v5.Context).(String|JSON|HTML|XML|Blob|Stream|File|NoContent|Redirect). ignore-sig-regexps leaves the default ignore-sigs (fmt.Errorf etc.) in place. golangci-lint config verify OK, golangci-lint run: 0 issues. Smoke test against in-memory DynamoDB Local: binary logged "found tables", started HTTP on :1323, wrote 5 metrics in ~3s, GET /metrics returned 200, bad from returned 400 "invalid from timestamp", SIGINT logged "Stopping..." and "Server shutting down..." and the process exited.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Moved the server/collector entrypoint from the repo root to cmd/tinymonitor/main.go (content unchanged) so a second binary can live under cmd/. Updated .vscode/launch.json to the new path. Fixed 4 wrapcheck errors that were already on master by ignoring echo response writers in .golangci.yml. Verified: go build, go vet, golangci-lint (0 issues), no .go files at root, and a runtime smoke test of collection, HTTP endpoint and graceful shutdown against DynamoDB Local.
<!-- SECTION:FINAL_SUMMARY:END -->
