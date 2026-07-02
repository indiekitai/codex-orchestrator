# LOOPS.md Contract / Restart / Context Follow-up

Date: 2026-07-02

Input:
- proxy: circulating `LOOPS.md: Field Notes on Agents That Run for Days` image
  and secondary excerpts.
- proxy: Tony Bai translation and Gist excerpt used only to understand the
  engineering ideas; this change does not claim an authoritative original
  source.

## Summary

The useful lesson is not a new product slogan. It is a tighter loop contract:
write the contract before implementation, write resume state to disk, allow a
bad worker to restart from the contract, and score subjective UI/workflow
quality with a rubric instead of vibes.

## Changes

- Added `codex-orchestrator context`, which writes a compact local/static resume
  pack for fresh sessions, context compaction, and long `/goal` runs.
- Expanded package specs with:
  - `Contract checklist`
  - `Restart policy`
  - `Subjective rubric`
- Added restart as a `pack eval` loop-control stop condition.
- Updated README, Chinese README, full guides, roadmap, research notes, and
  skill rules.
- Added tests for context-pack output and package spec required sections.

## Evidence Boundary

- proxy: source material was a circulating image and secondary references.
- local: Go helper and tests implement the new context/spec behavior.
- local/static: context packs summarize repo/ledger/status truth for humans and
  future Codex turns. They do not dispatch, merge, push, cleanup, deploy, or
  prove runtime/device/provider behavior.
- blocked: no claim is made about a real multi-day daemon, production runtime,
  or authoritative author source.

## Verification Plan

Run:

```sh
gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go
go test ./...
go run ./cmd/codex-orchestrator context --repo . --write-file /tmp/codex-orchestrator-context.md --write-report /tmp/codex-orchestrator-context.json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json --json
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --write-report /tmp/codex-orchestrator-docs-drift.json --json
git diff --check
```

## Verification Run

- passed: `go test ./...`
- passed: `go run ./cmd/codex-orchestrator context --repo . --write-file /tmp/codex-orchestrator-context.md --write-report /tmp/codex-orchestrator-context.json`
- passed: `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- passed: `go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json --json`
- passed: `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --write-report /tmp/codex-orchestrator-docs-drift.json --json`
- passed: `git diff --check`
