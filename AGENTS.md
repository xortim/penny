# AGENTS.md

Guidance for coding agents working in `github.com/xortim/penny`.

## Scope and precedence

- This repository is a Go Slack moderation bot built on Gadget.
- Follow explicit user instructions first, then this file, then existing code conventions.
- Repository-local instruction file exists at `CLAUDE.md`; align with it.
- No Cursor rules were found in `.cursor/rules/`.
- No `.cursorrules` file was found.
- No Copilot instructions were found at `.github/copilot-instructions.md`.

## Project map

- Entry point: `main.go` -> `cmd.Execute()`.
- CLI commands live in `cmd/` (Cobra + Viper).
- Main moderation behavior lives in `gadgets/hallmonitor/`.
- Slack/message helpers live in `pkg/conversations/`.
- Permalink parsing helpers live in `pkg/parsers/`.
- Build/version metadata lives in `conf/version.go`.

## Build, lint, and test commands

Prefer Make targets when available.

```bash
make help          # list targets
make build         # build binary to dist/<goos>-<goarch>/penny
make container     # build Docker image penny:local
make lint          # run golint
make test          # go test -coverprofile=coverage.out -covermode=atomic -v ./...
make all           # clean + verify + lint + test + build
make clean         # clear dist/ and coverage.out (+ modcache)
make tools         # install golint dependency
```

Direct Go commands used by maintainers:

```bash
go test -v ./...
go test -coverprofile=coverage.out -covermode=atomic -v ./...
```

### Run a single test (important)

Run one test function in one package:

```bash
go test -v -run '^TestPermalinkPathTS$' ./pkg/parsers
```

Run one test function in another package:

```bash
go test -v -run '^TestWhoReactedWith$' ./pkg/conversations
```

Run any matching test name (repo guidance pattern):

```bash
go test -v -run TestName ./pkg/parsers/
```

Re-run a package quickly without cache:

```bash
go test -v -count=1 ./pkg/parsers
```

## Local runtime and dependencies

- Bot server command: `penny serve` (alias: `penny server`).
- Default service port is `3000`.
- Local DB helper targets:

```bash
make start-db
make stop-db
```

- `make start-db` uses MariaDB 10.5 in Docker and expects DB env vars.
- Slack app setup/config examples are documented in `README.md`.

## Code style: Go conventions used in this repo

### Formatting and organization

- Always run `gofmt` on changed Go files.
- Keep imports grouped by standard library then external modules (gofmt/goimports style).
- Keep packages focused (`cmd`, `gadgets`, `pkg`, `conf`).
- Keep functions short and single-purpose when possible.

### Types and APIs

- Prefer concrete types unless an interface is required by the API boundary.
- Follow existing signatures that accept `slack.Client` for Slack interactions.
- Return zero values plus `error` on failures (`"", "", err`, `0, err`, etc.).
- Avoid introducing generics unless there is a clear repo-wide need.

### Naming

- Use `CamelCase` for exported identifiers and `camelCase` for unexported.
- Use descriptive names tied to Slack/domain language (`spamFeed`, `opMsgRef`, etc.).
- Existing code has some ALL_CAPS constants in `spam_feed.go`; do not rename without need.
- For new constants, prefer standard Go style (`BotMessageType`) unless matching nearby style is better.

### Error handling

- Return errors to callers when possible instead of panicking.
- Add context where useful (`fmt.Errorf("...: %w", err)`) for new code paths.
- Maintain behavior of user-facing Slack replies on recoverable failures.
- Do not silently swallow critical errors; at minimum log/print them consistently.

### Logging and output

- Repo currently mixes `println/print` and `zerolog`; prefer structured logging (`zerolog`) in new code.
- Keep log messages actionable and include channel/user/message identifiers when safe.
- Never log OAuth tokens, signing secrets, DB passwords, or other secrets.

### Configuration patterns

- Config is managed via Viper in `cmd/root.go` and `cmd/server.go`.
- Bind new flags through `viper.BindPFlag` and add env bindings where appropriate.
- Respect existing key namespaces (`slack.*`, `server.*`, `db.*`, `spam_feed.*`).
- Keep backward compatibility for existing keys/aliases if adding new config.

### Slack-specific behavior

- Message references and timestamps are sensitive; preserve timestamp handling logic.
- Do not change moderation thresholds/score semantics unless explicitly requested.
- Preserve threaded reply behavior and channel validation checks.
- Be careful with deletion paths (`DeleteMessage`) and only change with explicit intent.

## Testing conventions

- Tests are table-driven with `tests := []struct{...}` and `t.Run(...)` subtests.
- Existing assertions use `reflect.DeepEqual` and explicit `t.Errorf` messages.
- Keep tests deterministic; avoid real network/Slack calls in unit tests.
- Add tests next to code under `pkg/...` when changing parser/conversation logic.

## Agent workflow recommendations

- Before edits, read related files for local conventions.
- Make minimal, targeted changes; avoid broad refactors unless asked.
- If you touch behavior in `pkg/parsers` or `pkg/conversations`, run focused single tests first.
- For larger changes, run `make test`; run `make lint` when lint-sensitive changes were made.
- If command execution is unavailable, provide exact commands for the user to run.

## Safety and review checklist

- Do not commit secrets or sample credentials from docs/config.
- Keep backward compatibility for CLI flags/config unless user asks for breaking changes.
- Update documentation (`README.md`) when changing setup or runtime behavior.
- Verify build/test status before handing off substantial changes.
