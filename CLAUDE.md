# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Penny is a community moderation Slack bot for spam detection/removal, built in Go using the [Gadget](https://github.com/gadget-bot/gadget/) lightweight Slack bot framework. Named after Inspector Gadget's niece who "does all the work."

## Build & Development Commands

```bash
make build          # Build binary to dist/$(GOOS)-$(GOARCH)/penny
make test           # Run tests with coverage report
make lint           # Run golint
make all            # Full pipeline: clean, verify, lint, test, build
make tools          # Install golint dependency
make start-db       # Start local MariaDB 10.5 (needs DB_USER, DB_NAME, DB_PASS, DB_ROOT_PASS env vars)
make stop-db        # Stop local MariaDB
make container      # Build Docker image as penny:local
go test -v ./...    # Run all tests verbosely
go test -v -run TestName ./pkg/parsers/  # Run a single test
```

## Architecture

**Entry flow:** `main.go` → `cmd.Execute()` (Cobra CLI) → `penny serve` starts the bot on port 3000.

**Slack event handling:** Slack POSTs events to `/gadget` → Gadget framework routes to handlers in `gadgets/` → `hallmonitor` gadget processes spam reports.

**Key directories:**
- `cmd/` — CLI commands (root, server, version) using Cobra + Viper config
- `gadgets/hallmonitor/` — Core spam detection logic. `hallmonitor.go` registers routes; `spam_feed.go` contains the anomaly scoring and message removal flow
- `pkg/conversations/` — Slack message operations (retrieve, thread, react, mention)
- `pkg/parsers/` — Slack permalink parsing and timestamp conversion
- `conf/` — Build version metadata

**Spam detection flow in `spam_feed.go`:**
1. Reacji Channeler forwards flagged messages to a spam-feed channel
2. Bot parses the Slack permalink from the forwarded message
3. Calculates anomaly score: reported (2pts) + low activity (1pt) + outside timezone (2pts)
4. If score >= `max_anomaly_score`: deletes message and warns poster
5. Posts score breakdown and adds reaction emoji to spam-feed message

**Configuration:** Viper loads from `~/.penny.yaml`, env vars (prefix `PENNY_`), or CLI flags. Key config sections: `slack`, `server`, `db`, `spam_feed`. See README.md for full config reference.

## Interaction Style

When asked to plan something, present the options directly without asking clarifying questions first. Bias toward action over clarification.

## Build & Verify

Run `go build ./...` and `go test ./...` after any code changes to catch unexported type issues and compilation errors before committing.

## Testing

Tests use Go's table-driven testing pattern. Test coverage is in `pkg/` utilities only (`conversations_test.go`, `permalink_test.go`). No integration tests for Slack API interactions.

### Testing Conventions

When implementing tests, always use interfaces and dependency injection so handlers accept mock clients rather than creating their own API clients. Never instantiate real API clients (e.g., Slack, GitHub) inside handlers.

## Git & GitHub

For GitHub operations (PRs, issues, vulnerabilities), use the `gh` CLI tool directly rather than trying to access GitHub APIs programmatically.

## Branching

- `main` — production releases
- `develop` — active development (default working branch)
