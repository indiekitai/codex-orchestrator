# Eval Run Command Review

Date: 2026-06-11
Scope: V4 local/static eval runner command

## Result

Added `codex-orchestrator eval run` as a standalone fixture runner. It currently
supports the `orchestration-policy-auditor` suite and reads JSON fixtures from
`eval/orchestration-policy-auditor/` by default.

This separates two V4 use cases:

- `policy check`: scan the current repository and run fixtures.
- `eval run`: run fixtures only, useful when changing `OPA001`-`OPA005` rules.

## Evidence

### Local

- `go test ./...` passed.
- `go vet ./...` passed.
- `go run ./cmd/codex-orchestrator eval run --repo . --json` passed:
  - suite: `orchestration-policy-auditor`;
  - 6 fixtures passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` passed.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
  passed.
- `git diff --check` passed.
- `git diff --cached --check` passed.

### Blocked / Not Claimed

- This is local/static fixture evidence only.
- No automatic `eval add-failure` command exists yet.
- No `rules propose` command exists yet.
- No Codex App session, daemon, worker runtime, git mutation, or runtime proof
  was added.

## Self-Review

The command is intentionally read-only. It reuses the same fixture loader and
rule-count comparison as `policy check`, returns a normal `RoutineRunReport`,
and keeps unsupported suites blocked instead of silently passing.
