---
type: C4 Component
title: Host collector
status: stable
groma:
  id: collector
  parent: monitor-daemon
  code:
    - scanner: go
      file: internal/collector/collector.go
  technology: gopsutil v4
description: Samples host resource usage as metric records
---

Defines the `Collector` interface and its gopsutil implementation. `Memory()` reads virtual memory from the operating system and returns a `memory` metric with `total_gib`, `avail_gib`, and `used_pct` values, stamped with the current UTC time. The agent ID is a fixed placeholder (`123123123`), so every sample belongs to the same agent.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/collector/collector.go](../../../../../../internal/collector/collector.go) | [internal/model/metrics.go](../../../../../../internal/model/metrics.go) | Builds metric records | Go types |
| [internal/collector/collector.go](../../../../../../internal/collector/collector.go) | [host-operating-system](../../../../../externals/host-operating-system.md) | Reads memory statistics | gopsutil |
