# Routine Budget Alerts Observe/Heartbeat

Task: `TF-CODEX-ORCH-V4-ROUTINE-BUDGET-ALERTS-OBSERVE-HEARTBEAT-LOCAL`

## Scope

- Added helper-level budget pressure warnings to local `observe` and heartbeat output.
- Kept existing ledger compatibility: task `budget` remains optional and all new JSON fields are additive.
- Included routine spec budget counts by scanning repo-local `routines/*.json`.

## Evidence Labels

- `local`: Go unit tests use temporary ledgers and local git worktrees to verify budget metadata, pressure warnings, and heartbeat summary rendering.
- `local/static`: budget pressure is computed only from ledger timestamps, task observations, and routine spec JSON. It does not observe live Codex session runtime.
- `blocked`: exact review elapsed time remains unknown unless the ledger records a review-ready timestamp such as `completed-unreviewed`.

## Behavior

- `observe --json` and heartbeat JSON include additive `budgetPressure`.
- Heartbeat Markdown includes a `Budget Pressure` section when warnings exist.
- `status --json` includes `budgetSummary` so raw ledger status can be inspected without running `observe`.
- Runtime pressure uses the earliest recorded task timestamp.
- Review pressure uses the earliest recorded `completed-unreviewed` timestamp; if absent, the warning says elapsed review time is unknown.

## Boundaries

- No daemon-level scheduling.
- No task prioritization or dispatch changes.
- No process killing or budget enforcement.
- No package-manager, release, or publishing files changed.
