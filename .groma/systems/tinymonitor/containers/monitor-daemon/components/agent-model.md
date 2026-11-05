---
type: C4 Component
title: Agent model
status: stable
groma:
  id: agent-model
  parent: monitor-daemon
  code:
    - scanner: go
      file: internal/agent/agent.go
  technology: Go
description: Agent, job, and sink types that the daemon does not use
---

Declares an `Agent` with an ID, a collector, a sink, and a list of scheduled `Job`s, plus the `Sink` interface for storing metrics. No other package imports it: the daemon entry point runs its own fixed 500 ms memory loop, so these types are not part of the running daemon.
