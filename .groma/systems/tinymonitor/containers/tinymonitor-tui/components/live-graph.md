---
type: C4 Component
title: Live graph
status: stable
groma:
  id: live-graph
  parent: tinymonitor-tui
  code:
    - scanner: go
      file: internal/tui/model.go
    - scanner: go
      file: internal/tui/series.go
  technology: Bubble Tea v2, ntcharts v2 (timeserieslinechart)
description: Polls metrics and renders the rolling graph
---

The Bubble Tea model of the app. Each poll runs as a command outside the UI loop, and the next one is scheduled one refresh after the previous result, so requests never overlap and input never waits on the network. The first poll reads the whole window; later polls read only samples newer than the last one. Samples are kept as raw metrics, then turned into one series, split where data is missing, averaged into one point per braille column and drawn with padded Y bounds. A failed poll shows an error line while the last graph stays on screen; loading, no-data and unknown-value states replace the chart with a message.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/tui/model.go](../../../../../../internal/tui/model.go) | [internal/client/client.go](../../../../../../internal/client/client.go) | Polls metrics | Go interface (MetricsSource), Bubble Tea command |
