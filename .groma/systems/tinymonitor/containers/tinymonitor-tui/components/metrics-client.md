---
type: C4 Component
title: Metrics API client
status: stable
groma:
  id: metrics-client
  parent: tinymonitor-tui
  code:
    - scanner: go
      file: internal/client/client.go
  technology: Go net/http, JSON
description: Reads metrics from the daemon over HTTP
---

Wraps `GET /metrics`: it builds the query from an agent ID, a kind and an RFC 3339 time range, bounds each request with a timeout, and decodes the JSON array into metrics. A non-2xx answer becomes a StatusError carrying the status code and the start of the body. It accepts only http or https base URLs with a host.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/client/client.go](../../../../../../internal/client/client.go) | [internal/server/server.go](../../../../../../internal/server/server.go) | Queries metrics | HTTP GET /metrics, JSON |
