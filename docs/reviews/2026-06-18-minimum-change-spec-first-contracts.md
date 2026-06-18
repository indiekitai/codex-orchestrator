# 2026-06-18 Minimum-Change And Spec-First Contracts

## Trigger

Two current community projects showed useful adjacent patterns:

- Ponytail: make coding agents prefer reuse, deletion, standard library, and
  minimal implementation before writing new code.
- architect-loop: write specs and gates first, then use fresh-context builders
  and an architect/reviewer to judge evidence before integration.

This change imports the useful parts without changing codex-orchestrator's
positioning. The project remains a Codex App-first outer-loop harness, not a
fixed cross-vendor Fable/Codex architecture and not a personality layer.

## Changes

- Added a minimum-change gate to `SKILL.md`.
- Added worker contract language requiring a challenge step before
  implementation.
- Added self-review language for minimum-change rationale.
- Added feature-package planning questions for frozen spec/gate files.
- Updated English and Chinese full guides.
- Updated the roadmap to record the rule-level landing.

## Intended Behavior

Before implementation, a worker should now ask whether the task can be solved
by reuse, deletion, configuration, docs, tests, existing API/client/model, or a
smaller bounded change. If not, it should explain why the accepted contract is
still necessary and why the planned diff is the smallest coherent package step.

For important packages, the orchestrator should define outcome and gates before
dispatching builders. Gate/spec files are acceptance contracts and should not be
edited by workers unless that is the explicit task.

## Evidence

- `local/static`: runbook and documentation changes only.
- `blocked`: no helper auto-classifier was added. If real runs keep showing
  unnecessary new code or silent worker compliance with bad contracts, turn
  this into a policy/eval fixture or prompt-snippet generator.
