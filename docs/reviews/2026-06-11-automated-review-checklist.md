# Automated Review Checklist Local Slice

Date: 2026-06-11

Task ID: `TF-CODEX-ORCH-V4-AUTOMATED-REVIEW-CHECKLIST-LOCAL`

Evidence label: `local/static`

## Outcome

`run-routine pr-reviewer` now emits a conservative automated review checklist
for merge review of a ledger task. The checklist reuses the existing read-only
task/worktree/git inspection path and adds local/static signals before any
separate orchestrator merge decision.

The runner now checks:

- ledger task existence;
- worktree existence;
- expected branch match when recorded;
- dirty worktree status;
- commits after `baseCommit`;
- `git diff --name-status baseCommit..HEAD`;
- `git diff --check baseCommit..HEAD`;
- changed paths against ledger `writeSet.allowed` and `writeSet.forbidden` when
  recorded;
- locally detectable review artifact, artifact/report, self-review or handoff,
  and evidence-label filename signals;
- suggested narrow gates from the ledger task.

Forbidden-path hits and allowed-path misses fail the routine. Missing local
review/self-review/artifact/evidence filename signals remain warnings and set
`needsHuman`, because those signals may exist in the worker's final handoff
rather than in committed files.

## Boundary

This is not direct Codex App runtime, daemon, production, hardware, payment, or
device proof. The helper does not create sessions, merge, push, delete branches,
clean worktrees, run arbitrary gates, mutate the ledger, schedule work, or
replace human/orchestrator judgment.

## Residual Risk

The checklist is intentionally filename- and ledger-based. It can miss a
self-review or evidence label that exists only in chat, and it can warn on a
task where no review artifact was required. Treat the report as a merge-review
aid, not an acceptance decision.
