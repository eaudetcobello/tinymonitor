---
type: C4 Component
title: TUI entry point
status: stable
groma:
  id: tui-entrypoint
  parent: tinymonitor-tui
  code:
    - scanner: go
      file: cmd/tinymonitor-tui/main.go
  technology: Go
description: Starts the terminal app and maps its outcome to an exit code
---

Parses the command-line settings, builds the metrics API client, and runs the live graph as a Bubble Tea program in the alternate screen. Invalid flags exit with code 2 before the screen opens; a requested stop (q, Ctrl+C, SIGTERM) exits 0 and an app failure exits 1.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [cmd/tinymonitor-tui/main.go](../../../../../../cmd/tinymonitor-tui/main.go) | [internal/tui/config.go](../../../../../../internal/tui/config.go) | Parses flags | Go call |
| [cmd/tinymonitor-tui/main.go](../../../../../../cmd/tinymonitor-tui/main.go) | [internal/client/client.go](../../../../../../internal/client/client.go) | Creates the API client | Go call |
| [cmd/tinymonitor-tui/main.go](../../../../../../cmd/tinymonitor-tui/main.go) | [internal/tui/model.go](../../../../../../internal/tui/model.go) | Runs the graph | Bubble Tea program |
