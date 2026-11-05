---
type: C4 System
title: DynamoDB Local
description: Persistent storage for metric samples
status: stable
groma:
  id: dynamodb-local
  technology: Amazon DynamoDB (local endpoint)
---

The daemon reaches DynamoDB at the hard-coded endpoint `http://localhost:8000` in region `us-east-1` with static `local` credentials, so in practice this is DynamoDB Local running beside the daemon. tinymonitor owns the `metrics` table inside it: the daemon creates the table at startup (pay-per-request, partition key `agent_id`, sort key `sk` = `<kind>#<timestamp>`) and every item carries a `ttl` attribute about one year after the sample time.
