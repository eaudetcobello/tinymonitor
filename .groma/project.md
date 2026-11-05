---
type: Groma Project
title: tinymonitor
groma:
  profile: architecture
description: A small host monitor that samples memory usage, stores it in DynamoDB, serves it over HTTP and graphs it in the terminal
---

tinymonitor is a personal Go project. One daemon process samples the memory usage of the machine it runs on twice a second, writes each sample to a DynamoDB table, and exposes the stored time series through a read-only HTTP API. A separate terminal app, tinymonitor-tui, polls that API and draws a live graph. Start with the monitor-daemon container: its entry point wires the collector, the metrics store, and the HTTP API together. Then read the terminal UI container, which only talks to the daemon over HTTP.
