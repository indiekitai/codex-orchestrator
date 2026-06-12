# Project dashboard and package acceptance

## Scope

This change continues the feature-package closure work after the human-readable
status v2 slice. It focuses on making the helper more useful for real App-first
orchestration runs where the human wants to know which product lane is moving,
what has already closed, what needs review, and when an external model review is
worth running.

## Changes

- `status` / `observe` package rows now include:
  - `progressLabel`;
  - `humanSummary`;
  - `reviewStatus` from package-level external reviewer routine runs.
- `status --html` now renders package cards with a progress bar, human package
  summary, external review status, member task queues, and the package-specific
  next action.
- `status --write-summary` / Markdown package summary now includes progress and
  external review status.
- `pack review --package-id PKG` can now select all ledger tasks in that package
  when explicit `--task-id` flags are omitted.
- Added `pack acceptance --package-id PKG`, which aggregates task-level
  merge-readiness acceptance drafts and imported package external review runs
  into a single local/static package acceptance report.

## Evidence

- local/static: focused Go tests cover package dashboard fields, automatic
  package task selection, portable review pack generation, and package
  acceptance report generation.
- proxy/advisory: Pi read-only review reported no blocking findings. Follow-up
  fixes applied:
  - clarified `pack help` for package auto-selection;
  - made package reviewer status deterministic when multiple reviewer runs
    exist;
  - removed the unused acceptance-matrix decision parameter;
  - aligned live-proof gate wording with optional proxy/advisory reviewer
    signals;
  - added tests for wrong-package explicit task rejection and failed external
    reviewer attention.

## Boundaries

- No ledger schema migration is required.
- No Codex App session dispatch, merge, push, cleanup, daemon, launchd,
  production, device, payment, provider, or direct runtime proof behavior
  changed.
- External reviewer output remains `proxy/advisory`.
- Package acceptance reports remain `local/static`; they summarize evidence but
  do not authorize merge/push/cleanup without a separate orchestrator closeout
  decision.
