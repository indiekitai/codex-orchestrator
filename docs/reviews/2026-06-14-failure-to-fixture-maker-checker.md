# Failure-to-Fixture Maker/Checker Follow-up

Date: 2026-06-14

## Scope

Absorbed useful loop-engineering ideas from external agent/self-repair
examples without changing codex-orchestrator into a runtime tracing product.

## Changes

- Added `codex-orchestrator eval draft-failure`.
  - Reads `--text`, `--text-file`, or `--from-review`.
  - Runs the current orchestration policy rules read-only.
  - Reports expected versus actual `OPAxxx` rule hits.
  - Emits a suggested `eval add-failure` command only when counts match.
  - Does not write fixtures, mutate policy, dispatch workers, merge, push, or
    clean worktrees.
- Kept `eval add-failure` as the explicit approval/write step for regression
  fixtures.
- Updated starter templates with Maker/Checker and stop-condition fields.
- Updated README, full guide, v2 usage, routine docs, roadmap, and skill
  instructions to describe the failure-to-fixture flow.

## Evidence Labels

- local/static: CLI tests, docs updates, template updates, and policy/eval
  command behavior.
- blocked: no production trace harness, runtime agent observability platform, or
  automatic policy self-modification was introduced.

## Gates

- `go test ./...`: passed.
- `go run ./cmd/codex-orchestrator policy check --repo . --json`: passed,
  including 31 orchestration-policy-auditor fixtures.
- `go run ./cmd/codex-orchestrator validate-routines --json`: passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator eval draft-failure --repo . --id heartbeat-prompt-churn-demo --text "The orchestrator rewrote the heartbeat prompt on every wakeup with current task details." --expect OPA006=1 --json`:
  passed; produced a local/static draft and did not write a fixture.
- `git diff --check`: passed.

## Self-Review

- Maker/Checker boundary is explicit: drafting is not fixture acceptance.
- The new command is local/static only and cannot mutate eval fixtures.
- Existing `eval add-failure` remains the only fixture writer.
- Stop-condition fields were added to starter templates, not to project-specific
  private state.
- No Homebrew/npm/package-manager work was added.
