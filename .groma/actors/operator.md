---
type: C4 Actor
title: Operator
status: stable
groma:
  id: operator
---

A person watching how a host behaves right now. They start `tinymonitor-tui` in a terminal, pick the agent, metric kind, value and time window with flags, and read the live graph.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [operator](operator.md) | [cmd/tinymonitor-tui/main.go](../../cmd/tinymonitor-tui/main.go) | Watches metrics live | Terminal, command-line flags |
