---
type: C4 Component
title: Metrics store
status: stable
groma:
  id: metrics-store
  parent: monitor-daemon
  code:
    - scanner: go
      file: internal/service/metrics.go
  technology: AWS SDK for Go v2 (DynamoDB)
description: Persists metric samples in DynamoDB and queries them by time range
---

Owns the `metrics` table and its key design: partition key `agent_id`, sort key `<kind>#<timestamp>` with a fixed-width timestamp so lexical order equals time order. `CreateTable` creates the table if missing and waits up to 30 seconds for it. `Put` stores one sample with a TTL of 364 days. `Get` runs a paginated key query for one agent and kind between two times and returns samples in ascending time order. It depends only on a narrow `DynamoDB` interface, which keeps it testable without a live database.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/service/metrics.go](../../../../../../internal/service/metrics.go) | [internal/model/metrics.go](../../../../../../internal/model/metrics.go) | Maps stored items | Go types |
| [internal/service/metrics.go](../../../../../../internal/service/metrics.go) | [dynamodb-local](../../../../../externals/dynamodb-local.md) | Reads and writes items | DynamoDB API over HTTP |
