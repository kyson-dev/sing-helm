# Repository Guidelines

## Project Structure & Module Organization
- `cmd/minibox` hosts the CLI entry point and wiring for building the `minibox` binary.
- `internal/` holds reusable packages (core logic, proxy helpers, monitor UI, etc.); treat subdirectories as private APIs.
- `bin/` stores build artifacts (binaries, the `.minibox` dev home, logs, and state) so keep it ignored or regenerated via `make clean`.
- `docs/` collects supporting notes such as `proxy-hot-reload.md`, and `testdata/` houses fixtures used during unit tests.

## Build, Test, and Development Commands
- `make lint` runs `golangci-lint run` across the module to enforce Go best practices.
- `make test` (default) executes `go test ./... -cover` for a full suite with coverage.
- `make test-short` skips slow tests with `go test -short ./...`; use during rapid iteration.
- `make test-coverage` produces `coverage.out` and `coverage.html` to inspect metrics.
- `make build-dev` compiles `bin/minibox` with dev flags and populates `bin/.minibox` (`profile.json`, logs, state) for local experimentation.
- `make build-all` cross-compiles Linux, Windows, and macOS artifacts for release packaging.

## Coding Style & Naming Conventions
- Follow Go idioms: keep code `gofmt`-ed, prefer short, descriptive names, and document exported APIs.
- Binary and package names mirror their directories (`cmd/minibox`, `internal/daemon`).
- Rely on `golangci-lint` for formatting/naming catches following the repo tooling.

## Testing Guidelines
- Tests live alongside code under `*_test.go`; use `package foo_test` to keep boundaries clear.
- Rely on `go test ./...` for CI parity and `testdata/` for any deterministic fixtures.
- Tag slow cases with `t.Skip` or `build` tags if they need special flags, matching the existing `test-short` and `test-ci` targets.

## Commit & Pull Request Guidelines
- Keep commits short and type-prefixed like `feat: ...` or `fix: ...` as seen in history; focus on a single logical change per commit.
- PRs should describe the user impact, testing performed, and any manual steps (e.g., `make build-dev`).  Mention relevant issues or design notes when applicable.
- Attach before/after evidence for UX changes (logs/screenshots) since this project surfaces CLI+monitor details.
