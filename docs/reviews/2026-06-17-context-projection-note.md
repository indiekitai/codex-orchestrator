# Context Projection Note

Date: 2026-06-17

## Source

马东锡 NLP, "Context Is A Projection":
https://x.com/dongxi_nlp/status/2066991890348572950

## Summary

The useful idea is that a transcript records what happened, while context should
decide what matters for the next model call. For long Codex App orchestration,
the model should not receive the full old chat, giant logs, stale plans, and
completed worker details as one ever-growing blob.

## Codex-Orchestrator Implication

This maps directly to existing project direction:

- durable log: ledger, events, status JSON, review reports, artifacts;
- model-visible view: the short status/summary/context block used on heartbeat
  or resume;
- structured app state: package lane, run-mode, worker queues, gates, evidence
  labels, heartbeat gaps, and latest user override.

## Changes

- Added a `Context Projection` rule to `SKILL.md`.
- Added context projection analysis to `docs/research/loop-engineering-alignment.md`.
- Added a roadmap note under the package/status lane.

## Decision

Do not add a new helper command yet. Existing `status --write-summary`,
`status.html`, `health`, `preflight`, `thread-map`, `inbox`, and `concepts`
already cover part of this. A first-class `status --write-context` or
`.codex-orchestrator/context.md` should wait for more real-run evidence that
models still need a sharper handoff view.

## Evidence Labels

- `local/static`: docs and skill guidance only.
- `proxy`: the X article is practitioner framing, not direct product evidence.
- `blocked`: no runtime proof or Codex App context-management implementation
  was attempted.
- `direct`: none.

