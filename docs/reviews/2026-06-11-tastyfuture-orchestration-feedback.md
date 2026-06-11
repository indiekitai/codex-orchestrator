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
