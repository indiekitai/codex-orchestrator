# Status Page Human Mode Labels Review

Date: 2026-06-12

## Scope

Small usability enhancement for the local/static status surfaces:

- HTML status page
- Markdown status summary
- plain `status` / `observe` CLI output
- `status --json`

## Changes

- Added human-readable dispatch mode labels:
  - `active / 可继续派发`
  - `drain / 只收口，不派发`
  - `paused / 暂停编排`
- Added a dispatch-mode explanation block to the top human progress section.
- Made missed heartbeat warnings more visible in the HTML top section.
- Added `dispatchModeLabel` to `status --json`.
- Added tests to keep the human-readable wording from regressing.

## Evidence

- `local`: `go test ./...`
- `local`: `git diff --check`
- `local`: `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `local`: `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `local`: `go run ./cmd/codex-orchestrator status --repo . --html`
- `local`: `go run ./cmd/codex-orchestrator status --repo . --json`

## Boundary

This is local/static usability evidence only. It does not prove Codex App
heartbeat delivery, daemon behavior, production runtime, device behavior,
payment flow, provider integration, or hardware proof.

## Self-review

- Diff is limited to status rendering, tests, and this review note.
- No package-manager distribution, Homebrew, npm, tap, daemon, or session
  dispatch behavior was added.
- Evidence labels remain `local/static`; no local status signal is promoted to
  direct runtime proof.
