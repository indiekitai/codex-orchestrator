# 2026-06-12 TastyFuture Feedback Status Reconcile

## Scope

Addressed the latest TastyFuture orchestration feedback around status truth and
operator visibility:

- surface a real git worktree when ledger still records only a
  `pendingWorktreeId`;
- suppress generic dispatch recommendations while there is active or pending
  worker pressure;
- prefer the active package lane over old review debt in the human-readable
  status summary;
- show repo-local heartbeat/watchdog status in the HTML status overview.

## Evidence Label

local/static. These checks compare local ledger, git worktree, generated status,
and helper reports. They do not prove Codex App heartbeat delivery, OS wake
behavior, production state, device proof, payment, provider, or remote CI.

## Validation

- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `git diff --check`

## Self-Review

- Diff stayed inside the Go helper and review documentation.
- No Homebrew/npm/tap/package-manager work.
- No worker dispatch, merge, cleanup, or external side effect.
- Pending worktree reconciliation is read-only; it reports the mismatch and
  asks for `dispatch reconcile` instead of mutating ledger during `observe`.
- Heartbeat/watchdog output remains local/static and does not claim direct App
  automation proof.
