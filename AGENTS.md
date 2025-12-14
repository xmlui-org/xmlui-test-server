# Repository Guidelines

## Project Structure

- `cmd/`: CLI entrypoint (`main.go`) that builds the local server binary.
- `xmluisvr/`: primary Go library for request routing, DB access, config loading, and handlers.
- `test/`: integration-style tests and fixtures (`test-data/`); run via the workspace.
- `scripts/`: build/test/lint helpers used by the `Makefile`.
- `schemas/`: JSON schemas (versioned under `v1/`, `v2/`).
- `extensions/`: SQLite extension descriptors (e.g., Steampipe).
- `sql/`: SQL bootstrap/migrations used by local setups.
- `adrs/`: architecture decision records (design rationale).

## Build, Test, and Development

Use `make help` to see targets. Common workflows:

- `make deps` / `make tidy`: download deps and run `go mod tidy` across modules.
- `make fmt`: `gofmt -s -w .` (+ `goimports` if installed).
- `make lint`: runs `golangci-lint` across all Go modules.
- `make build`: builds the binary into `bin/` (default name is `xmlui-localsvr`; override with `BIN=xmlui-test-server`).
- `make run -- --help`: builds then runs the server, passing CLI args after `--`.
- `make test [dir]`: runs tests; optionally scope (e.g., `make test xmluisvr`).
- `make cover`: writes an HTML coverage report to `bin/coverage.html`.

This repo is a multi-module Go workspace (`go.work`); Go `1.25.x` is expected.

## Coding Style & Naming

- Go formatting is enforced via `gofmt`; keep imports gofmt/goimports-friendly.
- Package names: short, lowercase; exported identifiers: `PascalCase`.
- Tests: `*_test.go`, table-driven where it improves clarity.

## Testing Guidelines

- Run `make test` before opening a PR; use `make test <dir>` to iterate faster.
- Keep fixtures under `test/test-data/` and avoid network calls in tests.

## Commits & Pull Requests

- Prefer Conventional Commit-style subjects seen in history: `feat:`, `fix:`, `refactor:`, `chore:`, `ci:` (optional scope like `chore(cmd,test): ...`).
- PRs: describe behavior changes, include the exact command(s) run (e.g., `make test`), and link relevant ADRs/issues when applicable.

## Agent/Automation Notes

- Put scratch notes/specs in `~/` at the repo root (gitignored); avoid committing generated docs outside that folder unless explicitly intended.
