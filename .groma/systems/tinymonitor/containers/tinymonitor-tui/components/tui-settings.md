---
type: C4 Component
title: TUI settings
status: stable
groma:
  id: tui-settings
  parent: tinymonitor-tui
  code:
    - scanner: go
      file: internal/tui/config.go
  technology: Go flag package
description: Parses and validates the command-line flags
---

Turns `-server`, `-agent`, `-kind`, `-value`, `-window` and `-refresh` into a Config. Each flag is checked before the UI starts: the server URL with the metrics client rules, the kind with the shared metric model, and the refresh interval must be positive and no longer than the window. Errors name the offending flag; `-help` prints the usage.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/tui/config.go](../../../../../../internal/tui/config.go) | [internal/client/client.go](../../../../../../internal/client/client.go) | Validates the server URL | Go call |
| [internal/tui/config.go](../../../../../../internal/tui/config.go) | [internal/model/metrics.go](../../../../../../internal/model/metrics.go) | Validates metric kind | Go call |
