---
type: Groma Flow
title: Watch a metric live
groma:
  id: watch-a-metric-live
---

The operator starts the terminal UI; it polls the daemon every refresh interval and redraws the graph from the samples of the rolling window.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Operator](../actors/operator.md) | [TUI entry point](../systems/tinymonitor/containers/tinymonitor-tui/components/tui-entrypoint.md) | Start tinymonitor-tui with flags |
| [TUI entry point](../systems/tinymonitor/containers/tinymonitor-tui/components/tui-entrypoint.md) | [TUI settings](../systems/tinymonitor/containers/tinymonitor-tui/components/tui-settings.md) | Parse and validate the flags |
| [TUI entry point](../systems/tinymonitor/containers/tinymonitor-tui/components/tui-entrypoint.md) | [Live graph](../systems/tinymonitor/containers/tinymonitor-tui/components/live-graph.md) | Run the Bubble Tea program |
| [Live graph](../systems/tinymonitor/containers/tinymonitor-tui/components/live-graph.md) | [Metrics API client](../systems/tinymonitor/containers/tinymonitor-tui/components/metrics-client.md) | Fetch samples newer than the last one, off the UI loop |
| [Metrics API client](../systems/tinymonitor/containers/tinymonitor-tui/components/metrics-client.md) | [Metrics HTTP API](../systems/tinymonitor/containers/monitor-daemon/components/metrics-api.md) | GET /metrics for the agent, kind and range |
| [Metrics HTTP API](../systems/tinymonitor/containers/monitor-daemon/components/metrics-api.md) | [Metrics store](../systems/tinymonitor/containers/monitor-daemon/components/metrics-store.md) | Ask for samples in the validated range |
