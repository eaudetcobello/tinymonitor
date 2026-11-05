---
type: C4 System
title: Host operating system
description: Source of memory statistics for the machine the daemon runs on
status: stable
groma:
  id: host-operating-system
  technology: gopsutil
---

The collector asks the operating system for virtual memory totals through the gopsutil library, which reads the platform-specific kernel interfaces. tinymonitor only reads from it.
