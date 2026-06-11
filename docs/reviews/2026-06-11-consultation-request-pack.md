# Consultation Request Pack local/static review

Date: 2026-06-11

## Scope

Implemented a conservative helper command:

```bash
codex-orchestrator pack consultation --task-id TASK [--repo PATH] [--ledger PATH] [--write-report PATH] [--json]
```

The command turns a ledger task into a local/static consultation request for
blocked, stale, product-decision, or human/physical-action work. It reads the
local ledger, task history, routine-run records, recorded gates, evidence
labels, and local worktree metadata, then reports the inferred blocker,
attempted paths, required human input/action, decision options with tradeoffs,
next safe action, and whether to keep or clean the task branch/worktree.

## Evidence labels

- `local`: Go unit tests, local CLI smoke, Go build, local static docs, and JSON
  report generation from fixture ledger data.
- `proxy`: None.
- `direct`: None. This slice did not inspect runtime, production, device,
  payment, hardware, network, Codex App automation, or external-provider state.
- `blocked`: The actual product decision or human/physical action remains
  outside the pack.

## Boundaries

The helper is read-only except for an explicit `--write-report` JSON output. It
does not dispatch, merge, push, tag, cleanup worktrees, edit ledger state, edit
git state, call the network, or run task gates.

The pack is local/static consultation planning evidence only. It can format a
structured ask, but it cannot answer the ask or prove runtime/product/device
behavior.

## Commands run

Focused implementation checks:

```bash
go test ./cmd/codex-orchestrator -run 'TestPackConsultation|TestPackMergeReadiness|TestCompletionScriptsMentionCoreCommands'
```

Required final checks:

```bash
go test ./...
go build ./cmd/codex-orchestrator
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
go run ./cmd/codex-orchestrator pack consultation --repo /tmp/codex-orchestrator-consult-smoke/repo --ledger /tmp/codex-orchestrator-consult-smoke/repo/.codex-orchestrator/ledger.json --task-id SMOKE-CONSULT --json
git diff --check
```

## Residual risks

- Blocker and human-action detection are deterministic local metadata
  heuristics; ambiguous wording still needs human review.
- The command does not run recorded gates. It lists them so the orchestrator or
  reviewer can choose the next safe verification step separately.
- Ledger history or routine-run records can be incomplete or stale; the report
  labels that as local/static evidence rather than proof.

## Self-review

- Diff reread: CLI schema, command routing, completion text, tests, README docs,
  roadmap, and this review doc were reread after editing.
- Allowed paths: changes are limited to `cmd/codex-orchestrator/**`,
  `README.md`, `README.zh-CN.md`, and `docs/**`.
- Forbidden paths: no `.github/**`, `Formula/**`, `dist/**`, distribution
  files, release notes/tags, credentials, or unrelated project files changed.
- Docs drift: README, Chinese README, and roadmap now mention
  `pack consultation` and preserve local/static evidence boundaries.
- Verification gaps: no direct runtime/product/device/network proof was
  attempted or claimed.
