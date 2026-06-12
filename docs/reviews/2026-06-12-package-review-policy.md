# Package review policy

Date: 2026-06-12

## Summary

Added a local/static package review policy layer above the existing package
review pack and external reviewer runner.

This gives the Codex App orchestrator a deterministic way to answer:

- whether a package boundary needs external review;
- whether to run zero, one, or two reviewers;
- which reviewer commands are available locally;
- when manual import from DeepSeek, Claude, Pi, or a human reviewer is needed.

## Changes

- Added `docs/research/package-review-policy.md` as the durable design note.
- Added `codex-orchestrator review policy show`.
- Added `codex-orchestrator review policy check`.
- Added built-in default policy:
  - low risk: optional;
  - medium risk: one reviewer, default `pi`;
  - high risk: two reviewers, default `pi` + `claude`;
  - manual/import-only reviewers: `deepseek`, `human`.
- Added repo-local override path:
  `.codex-orchestrator/review-policy.json`.
- Documented the command in `README.md`, `README.zh-CN.md`, `SKILL.md`, and
  `docs/roadmap.md`.

## Evidence Labels

- `local/static`: policy decisions, command availability, and configuration
  loading are local/static evidence.
- `proxy/advisory`: actual external model output remains proxy evidence and is
  recorded separately by `review run` or `review import`.
- `direct`: none. The policy check does not run reviewers or prove runtime,
  device, provider, pre/prod, or payment behavior.

## Boundaries

- `review policy show/check` do not run external reviewers.
- They do not merge, push, cleanup, dispatch, deploy, or mutate git state.
- They do not make a package acceptance decision.
- `claude ultrareview` remains outside the default workflow.

## Gates

- `go test ./cmd/codex-orchestrator -run 'TestReviewPolicy|TestPackReview|TestReviewRun'`
- Full repository checks should be run before merge:
  - `go test ./...`
  - `go run ./cmd/codex-orchestrator policy check --repo .`
  - `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  - `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  - `git diff --check`

## Residual Risks

- Package-level ledger records are still future work. For now, package review
  state is represented through policy reports and `RoutineRun` entries with
  package/reviewer metadata.
- Risk classification is explicit input (`--risk`) in this MVP. Automatic risk
  inference from task write sets and roadmap package lanes remains a follow-up.
