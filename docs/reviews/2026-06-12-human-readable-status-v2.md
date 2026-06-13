# Human-readable status v2

## Scope

This change makes `codex-orchestrator status --html` and
`status --write-summary` more useful to a project owner who wants to know what
is happening without reading ledger internals.

## Changes

- Added a human-first `当前进度` panel before machine details.
- The panel answers:
  - current health;
  - current feature package lane;
  - recent completed work;
  - running or waiting work;
  - whether the human needs to act;
  - next safe step;
  - risk/evidence boundaries.
- Kept raw runtime, package, job, ledger, branch, and worktree details below the
  human summary.
- Humanized task/package identifiers by stripping common orchestration prefixes
  and preserving useful acronyms such as RBAC, API, POS, PAX, KDS, and UI.
- Updated tests so the first-screen summary no longer exposes dispatch-slot
  jargon or a conflicting "dispatch next task" recommendation while active work
  is still running.

## Evidence

- local/static: generated a restaurant POS rewrite status snapshot from a real ledger:
  - `/tmp/tf-status-v2.html`
  - `/tmp/tf-status-v2b.md`
- local/static: focused Go test passed:
  - `go test ./cmd/codex-orchestrator`
- proxy/advisory: Pi read-only review reported no blocking findings. Low-risk
  follow-ups were applied for evidence-label wording, the human-action section
  title, helper branch coverage, and a priority-order comment.

## Boundaries

- This is a rendering and documentation change.
- No ledger schema changed.
- No Codex App session dispatch, merge, push, cleanup, daemon, or runtime
  monitoring behavior changed.
- The status page remains `local/static` evidence. It is not direct proof of
  Codex App heartbeat delivery, pre/prod behavior, device state, payment state,
  or hardware behavior.
