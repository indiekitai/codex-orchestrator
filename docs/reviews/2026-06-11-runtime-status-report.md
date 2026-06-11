# Runtime Status Report

Date: 2026-06-11

This note records the bounded local/static helper work for the runtime status
report roadmap item. It is repository-local implementation evidence for
`codex-orchestrator`; it is not a claim of direct Codex App runtime, daemon,
production, payment, hardware, or human-review proof.

## Scope

- Extend the Go helper's existing `status` / `observe` / heartbeat report
  surfaces instead of adding a new controller or runtime.
- Show the current queue shape from local repo, ledger, and worktree truth.
- Keep the helper read-only with respect to session control, merge, push,
  worktree cleanup, and dispatch decisions.

## Delivered Surface

- `status` now prints a compact runtime snapshot for "what is happening now".
- `status --json`, `observe --json`, and heartbeat report JSON now expose a
  `runtimeStatus` object.
- The report groups tasks into local/static categories when possible:
  `activeWorkers`, `pendingSetup`, `dirtyUncommitted`,
  `completedUnreviewed`, `blockers`, `cleanupNeeded`,
  `recentMergedOrCleaned`, and `availableDispatchSlots`.

## Boundary

- Evidence label stays `local/static`.
- The helper still does not create sessions, merge, push, delete branches,
  delete worktrees, or claim direct runtime proof.
- "Recent merged/cleaned" is derived from local ledger task history timestamps,
  not from a live Codex App runtime API.
