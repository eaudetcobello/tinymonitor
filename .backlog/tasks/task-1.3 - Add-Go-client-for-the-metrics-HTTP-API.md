---
id: TASK-1.3
title: Add Go client for the metrics HTTP API
status: To Do
assignee: []
created_date: '2026-11-05T04:02:00Z'
labels:
  - api
  - tui
dependencies:
  - TASK-1.2
parent_task_id: TASK-1
priority: medium
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The TUI reads metrics over HTTP so it can run away from the server and never touches DynamoDB. A small typed client keeps HTTP and JSON details out of the UI code and can be tested without a terminal.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The client takes a base URL and returns `[]model.Metric` for (agent ID, kind, from, to)
- [ ] #2 Non-2xx responses come back as errors that include the status code and body
- [ ] #3 Requests honor context cancellation and have a timeout
- [ ] #4 Tests run against an `httptest` server and cover success, empty result and error responses
<!-- AC:END -->
