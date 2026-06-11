# Merge-Readiness Pack local/static review

Date: 2026-06-11

## Scope

Implemented a conservative helper command:

```bash
codex-orchestrator pack merge-readiness --task-id TASK [--repo PATH] [--ledger PATH] [--write-report PATH] [--json]
```

The command turns a ledger task into a standard review package for a
completed-unreviewed worker branch. It reads the local ledger and worker
worktree, then reports task metadata, git status, commit count after
`baseCommit`, `git diff --name-status`, allowed/forbidden path checks from the
ledger `writeSet`, `git diff --check`, review doc/artifact/self-review/
evidence-label/docs-drift signals, recorded gates, suggested gates, residual
risks, and `needsHuman` when evidence is incomplete.

## Evidence labels

- `local`: Go unit tests, Go build, local git diff inspection, local static docs
  updates, and generated JSON report shape.
- `proxy`: None.
- `direct`: None. This slice did not inspect Codex App runtime, worker session
  runtime, production, device, payment, hardware, or live user surfaces.
- `blocked`: Direct runtime/device/production proof is intentionally out of
  scope for this helper.

## Boundaries

The helper is read-only except for an explicit `--write-report` JSON output.
It does not merge, push, tag, cleanup worktrees, dispatch workers, edit ledger
state, edit git state, or run the worker task gates automatically.

The pack is local/static review evidence only. Missing review doc, artifact,
self-review, evidence-label, docs-drift, gate, worktree, branch, base commit,
or diff evidence is reported as `needsHuman`, `failed`, or `blocked` rather than
promoted to runtime or direct proof.

## Commands run

```bash
go test ./...
go build ./cmd/codex-orchestrator
gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go
```

Final checks for this branch:

```bash
go test ./...
go build ./cmd/codex-orchestrator
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
git diff --check
```

All final checks passed locally. The build command produced an untracked root
binary named `codex-orchestrator`; it was removed as a generated artifact.

## Residual risks

- The signal checks are filename/path heuristics. They deliberately require
  human review when evidence is absent or ambiguous.
- The pack suggests recorded gates and docs/evidence-label audit gates, but it
  does not run task-specific gates on behalf of the reviewer.
- The pack relies on the ledger `baseCommit` and `writeSet`; stale or incomplete
  ledger data remains a human review concern.
