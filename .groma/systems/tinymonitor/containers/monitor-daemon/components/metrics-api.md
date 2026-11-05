---
type: C4 Component
title: Metrics HTTP API
status: stable
groma:
  id: metrics-api
  parent: monitor-daemon
  code:
    - scanner: go
      file: internal/server/server.go
  technology: Echo v5, HTTP/JSON
description: Serves stored metrics over HTTP
---

An Echo server on `:1323` with request logging and panic recovery. Its only route, `GET /metrics`, binds `agent_id`, `kind`, `from`, and `to` query parameters, validates the kind and the RFC 3339 range (rejecting `from` after `to` with 400), and returns the matching samples as a JSON array. It reads through the `MetricsReader` interface, which the metrics store implements.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/server/server.go](../../../../../../internal/server/server.go) | [internal/service/metrics.go](../../../../../../internal/service/metrics.go) | Queries metrics | Go interface (MetricsReader) |
| [internal/server/server.go](../../../../../../internal/server/server.go) | [internal/model/metrics.go](../../../../../../internal/model/metrics.go) | Validates metric kind | Go call |
