---
type: C4 Container
title: Monitor daemon
status: stable
groma:
  id: monitor-daemon
  parent: tinymonitor
  technology: Go, Echo v5, AWS SDK for Go v2
description: Long-running process that collects, stores, and serves metrics
---

Built from `cmd/tinymonitor`. On start it connects to DynamoDB Local, ensures the `metrics` table exists, launches the HTTP API on `:1323` in a goroutine, and then runs the sampling loop on the main goroutine until it receives SIGINT or SIGTERM. Both the loop and the API share one metrics store, and shutdown cancels a shared context so the API drains gracefully within 10 seconds.
