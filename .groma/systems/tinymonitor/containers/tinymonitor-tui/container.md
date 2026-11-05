---
type: C4 Container
title: Terminal UI
status: stable
groma:
  id: tinymonitor-tui
  parent: tinymonitor
  technology: Go, Bubble Tea v2, Lip Gloss v2, ntcharts v2
description: Full-screen terminal app that graphs stored metrics live
---

Built from `cmd/tinymonitor-tui`. It runs on any machine that can reach the daemon, polls `GET /metrics` for one agent and metric kind, and draws the chosen value as a braille line graph over a rolling time window (5 minutes by default). It holds no state beyond the samples of the current window and never writes to the daemon.
