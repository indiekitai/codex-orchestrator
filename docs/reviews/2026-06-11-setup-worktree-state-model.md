# Setup/Worktree State Model Local Slice

Date: 2026-06-11

Task ID: `TF-CODEX-ORCH-V4-SETUP-WORKTREE-STATE-MODEL-LOCAL`

Evidence label: `local/static`

## Outcome

The helper now exposes setup/worktree state as structured local data instead of
requiring the orchestrator to infer it from chat or prose notes.

`observe`, `status --json`, and heartbeat reports keep the existing lifecycle
status buckets, then add a `state` object for each task observation and runtime
status item:

- `setup`
- `worktree`
- `branch`
- `diff`
- `review`
- `cleanup`

This keeps `pendingWorktreeId`, missing worktree paths, real worktrees, branch
mismatches, detached branches, dirty uncommitted diffs, clean task commits,
review-ready commits, cleanup-needed tasks, merged tasks, and cleaned tasks
distinct in helper output.

## Rules Captured

- `pendingWorktreeId` is a setup placeholder, not active work.
- Git/worktree truth wins over advisory thread status.
- A clean task commit after `baseCommit` is `completed-unreviewed` until
  orchestrator review.
- Dirty uncommitted work is separate from clean committed work.
- A detached worker worktree with an expected branch is `blocked`.
- Merged/released tasks with a remaining worktree are `cleanup-needed`.
- Cleaned terminal tasks are quiet and do not consume dispatch/review pressure.

## Boundary

This is not direct Codex App runtime, daemon, or session API proof. The helper
does not create sessions, merge, push, delete branches, delete worktrees, kill
workers, schedule work, or claim production/runtime proof.
