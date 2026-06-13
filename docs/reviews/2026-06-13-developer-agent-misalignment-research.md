# Developer-Agent Misalignment Research Review

Date: 2026-06-13

## Scope

Reviewed the developer-agent misalignment paper and the related practitioner
summary, then mapped the findings to `codex-orchestrator` product planning.

Changed files:

- `docs/research/developer-agent-misalignment.md`
- `docs/research/model-plateau-loop-engineering.md`
- `docs/roadmap.md`
- `docs/reviews/2026-06-13-developer-agent-misalignment-research.md`

## Findings

The paper validates a direction we have already seen in real orchestration
runs: the hard part is not only getting agents to write code. The daily pain is
keeping long-running agent work aligned with developer constraints, real repo
state, evidence labels, and honest progress reporting.

Product implication: the next `codex-orchestrator` feature package should be
developer-agent misalignment reduction, not another unrelated set of routine
commands.

Recommended queue:

1. Misalignment Event Log / Pushback Capture
2. Claim Verifier / Evidence-Bound Self-Report
3. Misalignment Taxonomy Policy/Eval Fixtures
4. Trust-Risk Status Block
5. Constraint Stack / Worker Contract Snapshot
6. Misalignment Insights Report

## Evidence Labels

- `local`: repo docs and roadmap were updated locally.
- `proxy`: the X post is a practitioner summary of the paper and was used as
  an interpretation signal, not primary evidence.
- `blocked`: no private transcript corpus was analyzed; no claim is made that
  the current tool detects all developer-agent misalignment cases.
- `direct`: none. This was research/planning work, not runtime proof.

## Verification

- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json` passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json` passed after rewording one roadmap example that looked like evidence promotion.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` passed.
- `git diff --check` passed.

## Self-Review

The update stays in documentation and planning surfaces only. It does not
change helper behavior, ledger schema, release packaging, or installed skills.
It does not claim direct proof, production proof, or model benchmark results.
