# Observe Reconcile And Active Heartbeat Guard

Date: 2026-06-13

## Context

A real App-first orchestration run completed successfully, but exposed two
state-management gaps:

- a worker had a clean task commit while the ledger still said `active`;
- an orchestrator deleted a heartbeat even though durable state still said the
  run was active for unattended continuation.

Both are local/static state problems. They do not prove task correctness or
runtime behavior, but they can cause missed reviews, stalled queues, or an
unattended run that silently stops.

## Change

- Added `codex-orchestrator observe --reconcile --write`.
- The reconcile path records deterministic git truth back to the ledger:
  resolved worktree, resolved branch, and `completed-unreviewed` when a clean
  task commit exists after `baseCommit`.
- Added policy/eval coverage for deleting a heartbeat while `dispatchMode` or
  run-mode remains active.
- Status Markdown now surfaces `latestUserOverride` from `dispatchNote` so
  humans can see the durable user intent next to raw capacity signals.
- Improved path matching for `**` across repository directories and surfaced
  `path <= pattern` matches in path-check output.
- Strengthened worker self-review guidance for cleanup/retry/event/outbox/
  lifecycle/migration/API tasks: repeated execution, idempotency, unique
  constraints, and duplicate side effects must be reviewed explicitly.
- Tightened evidence-scan guidance: check changed diff/added lines first, then
  separate positive proof claims from negative boundary statements.
- Updated skill and v2 docs with the new reconcile workflow and active
  heartbeat guard.

## Evidence Labels

- `local`: unit tests cover `observe --reconcile --write`.
- `local`: policy fixture covers active run-mode heartbeat deletion.
- `local`: unit test covers `**` allowed-path matching across directories and
  matched-rule reporting.
- `local/static`: reconciliation only records ledger/status truth.
- `blocked`: this does not review a worker diff, run task gates, merge, push,
  cleanup, deploy, or prove runtime/hardware/provider behavior.

## Operator Guidance

Use read-only observe for normal inspection:

```bash
codex-orchestrator observe --json
```

Use write reconcile only when observe/git truth has a deterministic ledger gap:

```bash
codex-orchestrator observe --reconcile --write --json
```

Do not delete or pause the App heartbeat while `dispatchMode=active` unless the
ledger has first been switched to `drain`/`paused`, or a concrete
queue-drained/blocker state has been recorded.
