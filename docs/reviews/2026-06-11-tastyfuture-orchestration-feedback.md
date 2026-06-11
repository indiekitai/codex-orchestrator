# TastyFuture Orchestration Feedback

Date: 2026-06-11

This note records feedback from a live TastyFuture Codex App orchestrator
session. It is local/project feedback for `codex-orchestrator`; it is not a
claim of direct runtime, production, payment, hardware, or daemon proof.

## Context

The TastyFuture orchestrator was running real delegated work across multiple
Codex App worktree sessions. It paused dispatch and reported only on observed
workflow behavior: worktree state, branch state, review/merge/cleanup flow,
evidence labeling, and recurring failure modes.

Evidence label: `local/project-feedback`.

## What Worked

- Worktree isolation turned each worker into a reviewable delivery unit instead
  of just another chat window.
- Repo truth checks such as `git status --short --branch`, `git worktree list
  --porcelain`, and recent commits prevented relying only on thread messages.
- Clean worker commits could be treated as `completed-unreviewed` even when the
  child thread state was ambiguous.
- Review, gates, merge, push, and cleanup formed a usable loop for real
  feature slices.
- Evidence labels were especially important for TastyFuture: many tasks were
  only `local`, `proxy`, or `blocked`, and should not be promoted to `direct`,
  `pre`, `prod`, `device`, or payment proof.

## Repeated Failure Modes

- Heartbeat prompts can become stale when they include specific task IDs instead
  of dynamically rediscovering repo, ledger, and worktree truth.
- An orchestrator can merge one child task and then stop, even though the larger
  queue should continue.
- `pendingWorktreeId` can be mistaken for an active worker before the real
  worktree and branch exist.
- Thread state and git state can disagree: a thread may look active while the
  branch has a clean commit, or a worktree may have useful uncommitted changes
  without a final self-review.
- Central docs such as progress or roadmap files can drift after worker merges
  if the orchestrator forgets to update them.
- Automation/heartbeat status is not visible enough to the user: last run, next
  run, blockers, and quiet waiting are hard to distinguish.
- External or human-action boundaries still rely heavily on prompt discipline:
  payment terminal, printer, device, network, DNS, pre/prod, and other
  real-world actions should be gated conservatively.
- Worker self-review quality is inconsistent and still needs reviewer
  inspection.

## Product Implications

The next valuable work is not adding more generic routines. It is making the
outer loop more observable and harder to misuse:

1. Runtime status report.
   Show active workers, pending setup, dirty-uncommitted work, completed
   unreviewed commits, merged-this-cycle, blockers, and available dispatch
   capacity.

2. First-class setup/worktree state model.
   Treat `pendingWorktreeId`, real worktree path, branch, dirty state, clean
   task commit, blocked state, merge state, and cleanup state as distinct
   states.

3. Automated review checklist.
   Inspect allowed and forbidden paths, review docs, artifacts, self-review,
   diff check, evidence labels, and docs drift before any merge decision.

4. Evidence-label linter.
   Keep `local`, `proxy`, `direct`, and `blocked` honest, especially in review
   docs, progress docs, and handoff summaries.

5. Post-merge docs drift guard.
   After accepted merges, report whether central progress/roadmap docs need an
   orchestrator-owned update or whether docs are explicitly not needed.

## Delta Feedback After Longer TastyFuture Run

Later TastyFuture orchestration added a sharper point: several earlier risks
are now covered by v0.3.0/V4 policy checks, but the live workflow still depends
too much on chat memory and manual judgment.

What continued to work:

- Repo truth still mattered more than child-thread messages. The orchestrator
  accepted completed work only after checking `git status --short --branch`,
  `git worktree list --porcelain`, worker branch commits, diffs, gates, and
  self-review.
- The review/merge/docs/push/cleanup sequence prevented two common failures:
  merged code without central docs and stale worktrees being mistaken for
  active workers.
- Evidence labels remained essential. Local readiness packages such as Admin
  compliance and Customer account/notifications could be treated as local
  readiness work, not production audit certification, live provider proof,
  IAM/RBAC proof, SMS/email/push proof, pre/prod proof, or device proof.

New or still-unresolved gaps:

- The ledger is not yet carrying the whole long-running state in practice.
  Pending ids, branch names, allowed paths, gates, review state, and whether a
  task has been accepted can still live primarily in heartbeat text or chat
  summaries.
- Central docs sync is still an orchestrator judgment call. Workers correctly
  avoid broad `PROGRESS.md` or roadmap edits, but the orchestrator needs a
  stronger project-aware docs drift input after merge.
- Next-task selection still depends on human/chat experience. When a roadmap is
  filled with local readiness pages and blocked live/provider/device work, the
  tool should help classify whether a candidate is vertical completion,
  runtime proof, blocker removal, owner-gated work, or shallow-risk readiness
  churn.
- Heartbeat status is still not transparent enough. The user should be able to
  see last trigger, next trigger, target thread, last action, and whether a
  quiet state means waiting, blocked, skipped, or no capacity.
- Orchestrator acceptance evidence is not a first-class artifact. Gates such as
  test/build/governance checks and `git diff --check` may be run, but the
  merge decision is not always stored as a machine-readable acceptance report.

Highest-priority improvements from this delta:

1. Ledger-enforced dispatch closure.
   Record task id, `pendingWorktreeId`, resolved thread, worktree, branch,
   base commit, allowed paths, gates, and status immediately after dispatch and
   reconciliation. Heartbeats should start from ledger truth instead of chat
   final messages.

2. Orchestrator acceptance report.
   After reviewing a worker, write a machine-readable report covering diff
   paths, forbidden path checks, docs drift decision, gates, evidence labels,
   merge decision, and residual risks.

3. Project-aware roadmap scorer.
   Support custom source-of-truth docs such as TastyFuture's `PROGRESS.md`,
   Chinese roadmap, and `docs/reviews/`. Candidate tasks should be classified
   as `vertical-completion`, `runtime-proof`, `blocked-removal`,
   `shallow-risk`, or `owner-gated`.

This delta does not overturn the existing V4 work. It narrows the next product
work from "more routines" to "ledger closure, acceptance artifacts, and
project-aware next-task scoring."

## Candidate Fixtures

- `pendingWorktreeId` exists but no worktree exists yet: should remain pending
  setup and must not dispatch a duplicate same-task worker.
- Worker thread appears active but branch has a clean task commit: should be
  `completed-unreviewed`.
- Worker rebase or proof rerun dirties artifacts after review: should report
  cleanup noise or require human confirmation before cleanup.
- Review doc writes local/proxy proof as direct/pre/prod proof: should fail.
- Worker changes outside allowed paths or touches central docs without explicit
  ownership: should fail or require review.

## Boundary

This feedback supports roadmap and policy/eval work. It does not prove a live
daemon, automatic session scheduler, direct Codex App API control, production
runtime behavior, payment behavior, or hardware behavior.
