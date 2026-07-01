# Loop Control Package Eval Follow-up

Date: 2026-07-01

Input:
- proxy: Yage AI article, "Loop Engineering: Senior Manager"
  https://yage.ai/share/loop-engineering-senior-manager-20260630.html

## Summary

This follow-up keeps the existing `codex-orchestrator` positioning. It does not
turn the project into a generic management platform or a fully autonomous
runtime.

The useful idea from the article is narrower: a mature engineering loop needs a
manager-level control surface that explains why the loop should continue,
stop, switch lanes, or block. That maps directly to package-level evaluation.

## Changes

- Added `loopControl` to `pack eval` reports.
- Added package spec sections for `Decision trace` and `SOP feedback`.
- Documented that `loopControl` is local/static guidance, not merge authority.
- Updated README, Chinese README, full guides, roadmap, and loop-engineering
  research notes.
- Added tests for:
  - stop-for-acceptance when required local/static layers pass;
  - continue-same-package when verifier/evidence layers are missing.

## Evidence Boundary

- proxy: the source article was used as framing input.
- local: Go code and tests implement the package evaluation behavior.
- local/static: `loopControl` recommends the next orchestration action from
  ledger/spec/worktree truth. It does not dispatch, merge, push, cleanup,
  deploy, or prove runtime/device/provider behavior.
- blocked: no claim is made about a production loop, external scheduler,
  device proof, or autonomous manager.

## Verification Plan

Run:

```sh
gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go
go test ./...
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json --json
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --write-report /tmp/codex-orchestrator-docs-drift.json --json
git diff --check
```
