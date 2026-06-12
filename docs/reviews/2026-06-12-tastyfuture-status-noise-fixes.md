# TastyFuture status noise fixes

## Scope

This change addresses three issues found during real TastyFuture
`codex-orchestrator` use:

- Chinese/non-ASCII paths could be quoted by Git and misread by path-boundary
  checks.
- `availableSlots` could look dispatchable while `run-mode=drain` or
  `run-mode=paused`.
- Untracked `.codex-orchestrator/` state files could make the integration
  checkout look dirty even when business code was clean.

## Changes

- `gitOutput` now runs Git with `core.quotePath=false`, so path checks and
  reports use human-readable non-ASCII paths.
- `IntegrationState` now separates:
  - business-code dirty status;
  - `.codex-orchestrator/` local state directory changes;
  - state-dir-only status.
- HTML and Markdown status now explain state-dir-only changes as local
  orchestration state, not business code dirtiness.
- HTML status renders dispatch slots as non-dispatchable when run mode is
  `drain` or `paused`, even if raw available slot count is greater than zero.
- Human-facing risk lines explicitly warn that raw `availableSlots` should not
  trigger new dispatch in `drain` or `paused`.
- `.gitignore` now ignores the whole `.codex-orchestrator/` local state
  directory, so generated ledger/status files do not appear as repo work.

## Evidence

- local/static: focused tests cover:
  - state-dir-only integration status;
  - drain dispatch-slot display;
  - non-ASCII path output with `core.quotePath=false`.
- local/static: Pi review found no blocking issues. Two low-risk polish notes
  were addressed: the Git helper now documents the `quotePath=false` choice, and
  the drain/paused slot label no longer uses the English `raw` wording.

## Boundaries

- No ledger schema migration is required.
- No Codex App session dispatch, heartbeat automation, daemon, worker control,
  merge, push, cleanup, production, device, payment, provider, or direct runtime
  proof behavior changed.
- These fixes are local/static helper visibility and path-classification
  improvements only.
