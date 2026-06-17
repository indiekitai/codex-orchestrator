# 2026-06-17 Night-Run Review Durability Feedback

## Trigger

A real project night run reported that external model review had been executed,
but the first closeout only left raw review artifacts under the local
`.codex-orchestrator/` control-plane directory. The project later needed a
committed summary under its durable review docs so a fresh orchestrator session
could recover why the package was accepted and what advisory risks remained.

## Findings

- `direct`: `.codex-orchestrator/` is intentionally ignored as local control
  state in this repository. That is still the right default for raw ledger,
  status, review-pack, and routine output noise.
- `local/static`: a local review pack is not enough when external review
  affects package acceptance, rejection, waiver, or closeout. Worktree cleanup,
  context compaction, or a fresh orchestrator session can otherwise lose the
  durable reason for the decision.
- `proxy/advisory`: external reviewer output can block or inform acceptance,
  but it does not authorize implementation, merge, push, cleanup, release,
  deploy, or direct runtime/device/provider proof.

## Rule Change

For important hands-off runs and feature-package closeout, the orchestrator must
commit a short project-level review summary when external review influenced the
decision. A good summary records:

- reviewer name or tool,
- review status,
- P0/P1/P2/P3 counts when known,
- accepted fixes or waivers,
- remaining advisory risks,
- the evidence label `proxy/advisory`.

Raw `.codex-orchestrator/` packs may remain local/untracked, but the closeout
reason must survive worktree cleanup and a fresh orchestrator session.

## Files Updated

- `SKILL.md`
- `docs/full-guide.md`
- `docs/full-guide.zh-CN.md`
- `docs/roadmap.md`
- `.gitignore`

## Evidence

- `local/static`: documentation and runbook updates only.
- `blocked`: no helper code change was made in this pass; the path-join failure
  observed in the project feedback should be addressed separately if reproduced
  against the helper.
