---
type: Groma Flow
title: Record a memory sample
groma:
  id: record-a-memory-sample
---

Every 500 ms the daemon entry point samples host memory and persists it. Failures at either step are logged and the next tick tries again.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Daemon entry point](../systems/tinymonitor/containers/monitor-daemon/components/daemon-entrypoint.md) | [Host collector](../systems/tinymonitor/containers/monitor-daemon/components/collector.md) | Request a memory sample on each tick |
| [Host collector](../systems/tinymonitor/containers/monitor-daemon/components/collector.md) | [Host operating system](../externals/host-operating-system.md) | Read virtual memory totals |
| [Daemon entry point](../systems/tinymonitor/containers/monitor-daemon/components/daemon-entrypoint.md) | [Metrics store](../systems/tinymonitor/containers/monitor-daemon/components/metrics-store.md) | Store the sample |
| [Metrics store](../systems/tinymonitor/containers/monitor-daemon/components/metrics-store.md) | [DynamoDB Local](../externals/dynamodb-local.md) | Put the item with its kind#timestamp sort key and TTL |
