# Evidence Label Linter Local Slice

Date: 2026-06-11

Task ID: `TF-CODEX-ORCH-V4-EVIDENCE-LABEL-LINTER-LOCAL`

Evidence label: `local/static`

## Outcome

`run-routine evidence-label-auditor` now has a narrower linter surface for the
restaurant POS rewrite evidence-promotion failure mode. It still runs read-only, but it now
also scans `docs/reviews/*.md` review and handoff notes and applies `ELA010` for
weak evidence wording promoted to direct, pre, prod, device, runtime, hardware,
or payment proof without explicit direct evidence wording.

The expanded scan is intended for repo-local documentation, routine reports,
routine specs, and ledger-shaped JSON only. It does not inspect real devices,
payment terminals, production, pre, browser runtime, or Codex App session
runtime.

## Rule Coverage

- `ELA001`-`ELA003`: routine spec evidence-description misuse.
- `ELA004`: weak evidence wording near strong proof wording.
- `ELA005`-`ELA008`: JSON/report-shape issues.
- `ELA009`: direct evidence recorded for routines that reserve direct proof.
- `ELA010`: local/static/proxy/weak evidence promoted to direct/pre/prod/device/
  runtime/payment proof without explicit direct evidence wording.

## Boundary

This is local/static evidence only. Findings are conservative suspicions for a
reviewer to inspect, not semantic proof of wrongdoing. The routine does not
stage, commit, merge, push, tag, release, create sessions, use subagents, use
Paseo, mutate the ledger, delete or clean worktrees, dispatch workers, or claim
direct runtime, production, pre, device, hardware, or payment proof.

## Residual Risk

The linter is deterministic text scanning. It can miss evidence promotion spread
across multiple sentences, and it can suppress lines that describe rule fixtures
or linter behavior. Treat the report as a review guard, not an acceptance
decision.
