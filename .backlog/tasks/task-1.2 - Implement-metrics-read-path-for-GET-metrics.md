---
id: TASK-1.2
title: Implement metrics read path for GET /metrics
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T04:18:00Z'
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
- [x] #1 `GET /metrics?agent_id=&kind=&from=&to=` returns the stored metrics for that agent and kind within [from, to], ordered by time ascending
- [x] #2 Missing or unknown `kind` returns 400 with a clear message
- [x] #3 `from` after `to` returns 400
- [x] #4 An empty range returns an empty JSON array, not null
- [x] #5 Results larger than one DynamoDB page are returned completely (pagination handled)
- [x] #6 JSON field names are stable snake_case (agent_id, kind, time, values)
- [x] #7 Tests cover the sort key range, pagination and parameter validation
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Model (internal/model/metrics.go)
   - Add json tags to model.Metric: agent_id, kind, time, values. Remove the unused SortKey field (only service.toItem builds the sk).
   - Add model.ParseKind(string) (Kind, error) that accepts the known kinds (memory, cpu) and errors otherwise. The TASK-1.3 client reuses model.Metric, so both sides share these tags.

2. Service (internal/service/metrics.go)
   - Replace DBClient *dynamodb.Client with an exported interface DynamoDB {CreateTable, DescribeTable, PutItem, Query}. *dynamodb.Client satisfies it, so cmd/tinymonitor/main.go does not change. This is what makes pagination testable without DynamoDB Local.
   - Add const tableName = "metrics" and use it in CreateTable, Put and Get.
   - Get(ctx, agentID, kind, from, to): Query with KeyConditionExpression "agent_id = :a AND sk BETWEEN :from AND :to", where :from/:to are "<kind>#" + time formatted with timestampFormat (UTC, fixed 9-digit nanos, so lexical order equals time order and BETWEEN is inclusive [from, to]). ScanIndexForward true for ascending order.
   - Iterate dynamodb.NewQueryPaginator until HasMorePages is false, UnmarshalListOfMaps into []metricItem, convert each to model.Metric (parse Time with timestampFormat). Start from make([]model.Metric, 0) so an empty range is [] and not null.

3. Server (internal/server/server.go)
   - Server depends on a small MetricsReader interface (Get) instead of *service.Metrics, so handler tests use a fake.
   - Split route setup into New(reader) *echo.Echo; Start keeps its signature and calls New. Tests drive New(...).ServeHTTP with httptest.
   - Add kind to GetMetricsParams. Validation, each a 400 with a plain text message like today: missing kind ("missing kind"), unknown kind ("unknown kind \"x\", expected one of: cpu, memory"), invalid from/to (existing), from after to ("from must not be after to").
   - Pass kind to Get and return the slice as JSON.

4. Tests (external test packages to satisfy the testpackage linter, t.Parallel everywhere)
   - service: fake DynamoDB whose Query returns two pages (LastEvaluatedKey set on the first). Assert the second call carries ExclusiveStartKey, the key condition and :from/:to values ("memory#2026-...000000000Z"), all items from both pages come back in order, and the empty-result case returns a non-nil empty slice.
   - server: table-driven httptest cases for missing kind, unknown kind, bad timestamps, from > to, empty result body "[]", and a success body with snake_case keys (agent_id, kind, time, values).

5. Verify: go build ./..., go test ./..., golangci-lint run. Optional manual check against DynamoDB Local on :8000 with curl.

Commits, one per logical change: model tags/ParseKind, service Get + interface, server kind param/validation, tests (or tests alongside each change).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Deviations from the plan:
- Added a tagliatelle json: snake rule to .golangci.yml, since the linter defaulted to camelCase and AC #6 requires snake_case.
- model.Kinds is a function rather than a package variable (gochecknoglobals).
- The test fake embeds service.DynamoDB and only implements Query.
- Service and server changes share one commit, because the Get signature change and its caller must land together for the build to pass.
- agent_id is still optional (missing means 0), as agreed.

Verification: go vet, go test ./... and golangci-lint run ./... all pass. DynamoDB Local was not running, so there was no manual curl check against a real table. Pagination is covered with a fake client that returns two pages.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
GET /metrics now takes agent_id, kind, from and to, and returns that agent's metrics of that kind within [from, to], in ascending time order, as snake_case JSON (agent_id, kind, time, values).

- Service: Metrics.Get queries 'agent_id = :agent_id AND sk BETWEEN "<kind>#<from>" AND "<kind>#<to>"'. Sort keys use a fixed-width UTC timestamp, so text order matches time order. It fetches every page through dynamodb.NewQueryPaginator and returns a non-nil empty slice when nothing matches. DBClient is now a small DynamoDB interface that *dynamodb.Client satisfies.
- Server: a missing or unknown kind, an invalid timestamp, or from after to each return a 400 with a plain-text message. The server depends on a MetricsReader interface, and routes are built in New() so handlers can be tested with httptest.
- Model: added JSON tags and ParseKind, and removed the unused SortKey field.

Tests cover the sort key range (including converting non-UTC bounds to UTC), following pages via ExclusiveStartKey, empty results, query errors, every validation path, the empty [] body and the JSON field names.

Follow-up: agent_id is not validated (missing means 0).
<!-- SECTION:FINAL_SUMMARY:END -->
