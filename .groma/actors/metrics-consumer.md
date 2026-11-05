---
type: C4 Actor
title: Metrics consumer
description: Reads recorded host metrics over HTTP
status: stable
groma:
  id: metrics-consumer
---

A person or tool that wants the time series tinymonitor has recorded. It calls `GET /metrics` on port 1323 with an agent ID, a metric kind, and an RFC 3339 time range, and receives the matching samples as JSON. The repository contains no client or dashboard; consumers call the API directly.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [metrics-consumer](metrics-consumer.md) | [internal/server/server.go](../../internal/server/server.go) | Queries metrics | HTTP GET /metrics, JSON |
