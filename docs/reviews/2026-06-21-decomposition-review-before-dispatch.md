# Decomposition Review Before Dispatch

Date: 2026-06-21
Scope: codex-orchestrator orchestration rules and public guide docs

## Context

Real project orchestration feedback showed a recurring risk: an orchestrator can
pick individually bounded worker splits that do not add up to one feature
package. The failure mode is visible in daily reports as scattered small tasks
instead of one package lane moving toward closure. `availableSlots` must stay a
capacity signal, not a dispatch reason.

## Decision

Add a coordinator-side decomposition review before worker dispatch. The
orchestrator must name:

- proposed worker list;
- dependency and merge order;
- same-package rationale;
- serial/shared-contract boundaries;
- allowed parallelism;
- gates and evidence labels;
- stop, drain, or package-switch condition.

If the split exists mainly because capacity is available, mixes unrelated
product lanes, or cannot be summarized as one feature-package advance, the
orchestrator must rewrite the plan before creating Codex App worker sessions.

## Boundary

This is a rule/docs update, not a new helper behavior. It is local/static
orchestration guidance and does not create sessions, mutate ledgers, merge,
push, deploy, or claim direct runtime proof.

## Docs Drift

Updated:

- `SKILL.md`
- `docs/full-guide.md`
- `docs/full-guide.zh-CN.md`
- `docs/roadmap.md`

No README positioning change was made.
