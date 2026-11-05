---
type: C4 Component
title: Metric model
status: stable
groma:
  id: metric-model
  parent: monitor-daemon
  code:
    - scanner: go
      file: internal/model/metrics.go
  technology: Go
description: Shared metric record and metric kind vocabulary
---

Defines `Metric` (agent ID, kind, time, and named float values, with snake_case JSON tags) and the `Kind` enumeration (`memory`, `cpu`). `ParseKind` turns user input into a known kind and returns `ErrMissingKind` or `ErrUnknownKind` otherwise. The collector, metrics store, and HTTP API all exchange data in this shape.
