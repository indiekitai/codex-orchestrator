# Pending Worktree Ledger Support Review

Date: 2026-06-11

## Scope

This pass adds project-local ledger support for Codex App pending worktree setup
IDs. The helper stores `pendingWorktreeId` as an opaque string so a task can be
visible in `observe` before the corresponding worktree exists.

## Expected Behavior

- `record-task --pending-worktree-id ID` records a task without requiring a
  worktree path.
- Pending tasks are classified as `pending-setup`, not as missing-worktree
  blockers.
- `append-event --worktree PATH --branch BRANCH` can reconcile the same task
  after setup finishes.
- `observe --json`, heartbeat reports, Markdown summaries, and `status --json`
  expose `pendingWorktreeId` when present.

## Boundaries

The helper still does not create Codex sessions, query Codex App APIs, create
worktrees, merge, push, delete branches, or clean up worktrees. The pending ID is
only local ledger metadata.

## Verification Plan

- `go test ./...`
- Manual smoke with a temp ledger: record pending task, observe pending, append
  setup-complete with worktree/branch, observe active.
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `git diff --check`
- `git diff --cached --check`
