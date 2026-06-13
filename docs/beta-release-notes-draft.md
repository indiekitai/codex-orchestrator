# v0.3.7 Release Notes

`v0.3.7` is a real-run reliability release for the Codex App-first
orchestration workflow. It focuses on the problems found while running
long-lived project orchestration loops: stale ledger state, heartbeat deletion
while the queue is still active, path-matcher false positives, and noisy
evidence scans.

## Highlights

- Added deterministic ledger catch-up:
  - `codex-orchestrator observe --reconcile --write --json`.
  - Records resolved worktree/branch and moves clean task commits to
    `completed-unreviewed`.
- Added a policy/eval guard against deleting or stopping the heartbeat while
  durable run-mode or `dispatchMode` remains active.
- Improved allowed/forbidden path matching:
  - `**` now matches across repository directory segments.
  - review reports show `path <= pattern` for matched rules.
- Strengthened worker self-review guidance for cleanup, retry, event/outbox,
  lifecycle API, migration, aggregate-version, and unique-constraint changes.
- Tightened evidence-scan guidance so reviewers check changed diff/added lines
  first and separate positive proof claims from negative boundary statements.
- Updated the installed skill and v2 docs with the new reconciliation workflow
  and active heartbeat guard.

## Why This Release

Real orchestration runs exposed a practical gap: the Git worktree can already
contain a clean task commit while the durable ledger still says the task is
active. Without a safe reconciliation command, the orchestrator has to remember
to manually repair state before review.

Another real issue was heartbeat lifecycle control. A continuous queue should
not silently stop just because the current child batch is empty when the ledger
or latest user instruction still says the run should continue unattended.

`v0.3.7` turns those lessons into helper behavior, skill rules, and regression
fixtures.

## New / Changed Commands

Reconcile deterministic local git truth into the ledger:

```bash
codex-orchestrator observe --reconcile --write --json
```

This can record:

- the real worktree path for a pending/active task;
- the real branch;
- `completed-unreviewed` when a clean task commit exists after `baseCommit`.

It does not accept work, merge, push, cleanup, deploy, or prove runtime
behavior.

`policy check` now includes the active heartbeat deletion regression fixture.

## Evidence Boundary

All new surfaces are `local/static` evidence. They are state-management and
review-assistance controls, not direct runtime proof.

Use `observe --reconcile --write` only after read-only observe/git truth shows
a deterministic ledger gap. After reconciliation, still run normal review:
diff, allowed/forbidden paths, self-review, docs/reviews, gates, evidence
labels, and `git diff --check`.

## Verification Before Publishing

Checks used for this release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator policy check --repo .`
- `go run ./cmd/codex-orchestrator validate-routines --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `/Users/tf/.local/bin/codex-orchestrator policy check --repo .`
- `git diff --check`

The local helper was rebuilt and the installed Codex skill was synced before
publishing.

## Suggested Announcement

`codex-orchestrator v0.3.7` hardens real Codex App orchestration loops: ledger
state reconciliation, active heartbeat deletion guards, better path matching,
and stronger review guidance for idempotency and evidence claims.
