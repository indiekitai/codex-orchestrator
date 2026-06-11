# Package external review runner

Date: 2026-06-12

## Summary

Added a package-level external review workflow for codex-orchestrator.

The goal is not to run another model on every small worker. The intended trigger
is a feature package boundary: several related slices, shared contract/API/DB
risk, security/payment/hardware/pre/prod boundaries, or a user-facing outcome
that deserves a second review pass.

## Changes

- Added `pack review` to build a portable local/static review pack from one or
  more ledger tasks.
- Added `review run` for read-only local reviewer execution with supported
  `pi` and `claude -p` runners.
- Added `review import` to record external review output from DeepSeek, Claude,
  Pi, or human reviewers into the ledger.
- Extended routine run records with optional `packageId`, `reviewer`, and
  `reportPath` fields.
- Documented package-level review in `README.md`, `README.zh-CN.md`,
  `SKILL.md`, and `docs/roadmap.md`.

## Evidence Labels

- `local`: review packs are generated from local ledger and git truth.
- `proxy`: external model and human review output is advisory evidence.
- `blocked`: reviewer command failure or missing review setup blocks the review
  record, not the implementation by itself.
- `direct`: none. This feature does not produce runtime, device, provider,
  production, or live proof.

## Boundaries

- No `claude ultrareview` path is included by default.
- External reviewer output does not authorize implementation, merge, push,
  cleanup, release, deploy, or direct runtime/device/provider proof.
- `review run` is intended to invoke reviewers in read-only mode.
- `review import` records a review result; it does not decide acceptance.

## Gates

- `go test ./cmd/codex-orchestrator -run 'TestPackReview|TestReviewRun|TestPackMergeReadinessWritesStandardLocalReport'`
- Full repository checks should be run before merge:
  - `go test ./...`
  - `go run ./cmd/codex-orchestrator policy check --repo .`
  - `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  - `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  - `git diff --check`

## Residual Risks

- `pi` and `claude -p` command-line behavior can vary by local installation.
  The runner records failures as blocked/proxy review setup issues.
- Package selection still depends on orchestrator judgment. The workflow records
  package review status, but does not yet automatically decide which package
  boundary deserves an external review.
