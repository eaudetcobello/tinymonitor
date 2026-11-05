---
type: Groma Flow
title: Query stored metrics
groma:
  id: query-stored-metrics
---

A consumer asks for one agent and metric kind over a time range and receives the samples in ascending time order.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Metrics consumer](../actors/metrics-consumer.md) | [Metrics HTTP API](../systems/tinymonitor/containers/monitor-daemon/components/metrics-api.md) | GET /metrics with agent_id, kind, from, and to |
| [Metrics HTTP API](../systems/tinymonitor/containers/monitor-daemon/components/metrics-api.md) | [Metric model](../systems/tinymonitor/containers/monitor-daemon/components/metric-model.md) | Parse and validate the kind |
| [Metrics HTTP API](../systems/tinymonitor/containers/monitor-daemon/components/metrics-api.md) | [Metrics store](../systems/tinymonitor/containers/monitor-daemon/components/metrics-store.md) | Ask for samples in the validated range |
| [Metrics store](../systems/tinymonitor/containers/monitor-daemon/components/metrics-store.md) | [DynamoDB Local](../externals/dynamodb-local.md) | Run a paginated key query between the two sort keys |
