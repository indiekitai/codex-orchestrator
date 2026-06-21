# AI-Native Team Operating Rules

Date: 2026-06-21
Scope: codex-orchestrator orchestration rules and guide docs

## Source Insight

A Claude Code / Cowork team operations summary emphasized that higher coding
throughput shifts the bottleneck toward validation, quality signals, feedback
intake, and process hygiene. The useful lesson for this repository is not to
chase more autonomous workers, but to make orchestration progress easier to
judge.

## Applied Changes

- Added `bad` / `sad` orchestration quality signals:
  - `bad`: severe failures such as wrong merge, forbidden-path merge, evidence
    promotion, hidden setup failure, or reviewed commits left unpushed without a
    blocker.
  - `sad`: recoverable friction such as stale status, helper false positives,
    confusing dispatch slots, reviewer timeouts, or unclear package rows.
- Added progress-not-motion discipline: worker count, task count, token use,
  command volume, and cleaned count are not progress unless they map to package
  outcome, blocker removal, evidence-level advance, accepted review, or clean
  closeout.
- Tightened Inbox role: feedback, social posts, real-run retrospectives, helper
  false positives, and reviewer notes should be triaged before becoming worker
  tasks.
- Added process hygiene guidance: remove, hide, or demote workflow surfaces that
  no longer help the orchestrator dispatch, review, merge, block, or stop.

## Boundary

This is a local/static rules and docs update. It does not add a daemon, change
helper runtime behavior, create sessions, mutate ledgers, merge, push, deploy,
or claim direct proof.

## Docs Drift

Updated:

- `SKILL.md`
- `docs/full-guide.md`
- `docs/full-guide.zh-CN.md`
- `docs/roadmap.md`
