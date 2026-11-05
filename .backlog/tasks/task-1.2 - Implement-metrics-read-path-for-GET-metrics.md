---
id: TASK-1.2
title: Implement metrics read path for GET /metrics
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - api
dependencies: []
references:
  - internal/service/metrics.go
  - internal/server/server.go
parent_task_id: TASK-1
priority: high
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`service.Metrics.Get` is a stub returning nil, so `GET /metrics` always returns null and nothing can be graphed. Items are keyed by `agent_id` (hash) and `sk = "<kind>#<timestamp>"` (range), so a time range query is only efficient per kind, but the endpoint has no kind parameter today.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `GET /metrics?agent_id=&kind=&from=&to=` returns the stored metrics for that agent and kind within [from, to], ordered by time ascending
- [ ] #2 Missing or unknown `kind` returns 400 with a clear message
- [ ] #3 `from` after `to` returns 400
- [ ] #4 An empty range returns an empty JSON array, not null
- [ ] #5 Results larger than one DynamoDB page are returned completely (pagination handled)
- [ ] #6 JSON field names are stable snake_case (agent_id, kind, time, values)
- [ ] #7 Tests cover the sort key range, pagination and parameter validation
<!-- AC:END -->
