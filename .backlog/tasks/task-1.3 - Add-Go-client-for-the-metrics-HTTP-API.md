---
id: TASK-1.3
title: Add Go client for the metrics HTTP API
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T04:02:00Z'
updated_date: '2026-11-05T04:26:00Z'
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
- [x] #1 The client takes a base URL and returns `[]model.Metric` for (agent ID, kind, from, to)
- [x] #2 Non-2xx responses come back as errors that include the status code and body
- [x] #3 Requests honor context cancellation and have a timeout
- [x] #4 Tests run against an `httptest` server and cover success, empty result and error responses
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
0. Branch: create task-1.3-metrics-client from task-1.2-metrics-read-path (task branches are stacked, master is still at the first commit).

1. Package internal/client (new file internal/client/client.go)
   - The TUI lives in the same module, so internal/ is enough.
   - New(baseURL string, timeout time.Duration) (*Client, error): parse with url.Parse, reject URLs without an http/https scheme or a host so a bad --server flag fails at startup (TASK-1.4 AC #3). Builds its own http.Client{Timeout: timeout}; a timeout <= 0 falls back to 10s.
   - (c *Client) Get(ctx, agentID uint32, kind model.Kind, from, to time.Time) ([]model.Metric, error). Same signature as server.MetricsReader, so the TUI can depend on a small interface and use a fake in its own tests.
   - Request: url.JoinPath(base, "metrics") so a trailing slash or a path prefix on the base URL works; query agent_id, kind, from, to with from/to as UTC RFC3339Nano (what the handler parses). http.NewRequestWithContext so ctx cancellation aborts the request (noctx linter); http.Client.Timeout bounds it when the ctx has no deadline.
   - Non-2xx: return *StatusError{StatusCode int, Body string}, body read through io.LimitReader (4 KiB) and trimmed. Error() reads "unexpected status 400: missing kind". Exported so callers can errors.As on it.
   - 2xx: json.NewDecoder into []model.Metric; a null body comes back as an empty non-nil slice, matching the server. Transport and decode errors are wrapped with context (wrapcheck).
   - No retries: TASK-1.5 owns retry/backoff at the polling layer.

2. Tests (internal/client/client_test.go, package client_test, t.Parallel)
   - Success: httptest server asserts method, path /metrics and each query param (from/to given in a non-UTC zone come out as UTC RFC3339Nano); returned metrics decode with time, kind and values intact.
   - Empty result: [] body returns a non-nil empty slice and no error.
   - Error responses: table of 400 (plain-text body) and 500; errors.As yields *StatusError with the status code and body, and the error string contains both.
   - Context cancellation: handler blocks on r.Context().Done(); cancelling the caller ctx returns an error wrapping context.Canceled.
   - Timeout: client with a short timeout against a handler that blocks; Get returns an error well before the handler would answer.
   - Malformed JSON body returns a decode error.
   - New rejects "", "localhost:1323" (no scheme) and "ftp://x"; accepts a base URL with a trailing slash and with a path prefix (request lands on <prefix>/metrics).
   - Contract test: run server.New(fake reader) under httptest.NewServer and call client.Get against it, so a parameter name or JSON tag mismatch between both sides fails a test.

3. Verify: go build ./..., go vet ./..., go test ./..., golangci-lint run ./...

Commits: one for the client package with its tests, one for the contract test if it ends up separate.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Deviations from the plan:
- New also exports ErrInvalidBaseURL so callers (TASK-1.4 flag validation) can errors.Is on it.
- Body.Close is wrapped in a deferred func to satisfy errcheck.
- The contract test is split in two (matching params and JSON, and an unknown-kind 400) to stay under the cyclop limit. Client and tests share one commit.

Verification: go build, go vet, go test -race ./... and golangci-lint run ./... (0 issues) all pass.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Adds internal/client, a typed Go client for GET /metrics that the TUI (TASK-1.5) uses instead of raw HTTP.

- client.New(baseURL, timeout) rejects base URLs without an http/https scheme or a host (ErrInvalidBaseURL). A timeout <= 0 uses DefaultTimeout (10s).
- Client.Get(ctx, agentID, kind, from, to) has the same signature as server.MetricsReader. It sends from/to as UTC RFC3339Nano, joins /metrics onto any base path, and returns a non-nil empty slice for [] or null.
- Non-2xx responses return *StatusError{StatusCode, Body} with at most 4 KiB of the trimmed body. Requests use the caller's context and the client timeout.

Tests use httptest: query and decoding, empty results, 400/404/500, body truncation, malformed JSON, cancellation, timeout, base URL validation and path joining, plus a contract test against the real server.New handler.

Follow-up: none. Retries stay with the polling loop in TASK-1.5.
<!-- SECTION:FINAL_SUMMARY:END -->
