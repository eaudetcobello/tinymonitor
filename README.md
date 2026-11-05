# tinymonitor

Collects CPU and memory samples, stores them in DynamoDB, serves them over HTTP and graphs them in the terminal.

## Setup

Requirements: Go (see `go.mod`) and Docker.

```sh
git clone git@github.com:eaudetcobello/tinymonitor.git
cd tinymonitor

# DynamoDB Local on :8000 (tinymonitor creates its table on start)
docker run -d -p 8000:8000 amazon/dynamodb-local

# Collector and API on :1323
go run ./cmd/tinymonitor

# In another terminal: the live graph
go run ./cmd/tinymonitor-tui
```

Run `go run ./cmd/tinymonitor-tui -help` for the TUI flags.

## Checks

```sh
go test ./...
golangci-lint run
```

## Getting changes into master

`master` only changes through pull requests.

```sh
git switch master && git pull
git switch -c my-change
# edit, then commit each logical change separately
git push -u origin my-change
gh pr create --base master --fill
gh pr merge --merge --delete-branch
```
