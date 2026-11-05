---
type: C4 Component
title: Daemon entry point
status: stable
groma:
  id: daemon-entrypoint
  parent: monitor-daemon
  code:
    - scanner: go
      file: cmd/tinymonitor/main.go
  technology: Go
description: Wires the daemon together and runs the sampling loop
---

Creates the DynamoDB client (fixed local endpoint and credentials, failing fast if `ListTables` does not answer), asks the metrics store to create its table, and starts the HTTP API. It then ticks every 500 ms: each tick asks the collector for a memory sample and hands it to the metrics store. Collection and storage errors are logged and the loop continues with the next tick.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [cmd/tinymonitor/main.go](../../../../../../cmd/tinymonitor/main.go) | [internal/collector/collector.go](../../../../../../internal/collector/collector.go) | Samples memory | Go call |
| [cmd/tinymonitor/main.go](../../../../../../cmd/tinymonitor/main.go) | [internal/service/metrics.go](../../../../../../internal/service/metrics.go) | Stores samples | Go call |
| [cmd/tinymonitor/main.go](../../../../../../cmd/tinymonitor/main.go) | [internal/server/server.go](../../../../../../internal/server/server.go) | Starts the API | Go call, goroutine |
| [cmd/tinymonitor/main.go](../../../../../../cmd/tinymonitor/main.go) | [dynamodb-local](../../../../../externals/dynamodb-local.md) | Checks connectivity | DynamoDB ListTables |
