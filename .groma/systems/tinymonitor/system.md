---
type: C4 System
title: tinymonitor
status: stable
groma:
  id: tinymonitor
description: Records host memory metrics and serves them as time series
---

tinymonitor measures resource usage on the host it runs on and keeps the history so it can be queried later. Today it collects memory statistics only (CPU is a declared metric kind with no collector), runs as a single daemon, and persists samples in DynamoDB Local. Consumers read the history through `GET /metrics`.
