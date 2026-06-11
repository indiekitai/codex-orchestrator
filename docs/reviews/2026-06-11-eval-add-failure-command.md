# Eval Add Failure Command Review

Date: 2026-06-11
Scope: V4 manual eval fixture creation command

## Result

Added `codex-orchestrator eval add-failure` as a manual way to turn a known
orchestration failure text into a checked eval fixture.

The MVP supports:

- `--suite orchestration-policy-auditor`;
- `--repo` and optional `--eval-dir`;
- `--id` as the JSON fixture filename;
- `--file` as the synthetic file path inside the fixture;
- `--text` or `--text-file`;
- repeated `--expect RULE=N`;
- `--force` for explicit overwrite.

Before writing, the command runs the text through the current policy rules and
requires actual rule-hit counts to match `--expect`. Mismatches fail without
writing a fixture.

## Evidence

### Local

- `go test ./...` passed.
- `go vet ./...` passed.
- A temporary `eval add-failure` run wrote one fixture outside the repository.
- `eval run` passed against that temporary fixture directory.
- `go run ./cmd/codex-orchestrator eval run --repo . --json` passed.
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

- The command does not parse review documents automatically.
- The command supports only the `orchestration-policy-auditor` suite.
- The command writes local fixture JSON and does not dispatch sessions, mutate
  ledgers, merge, push, release, or claim runtime proof.

## Self-Review

The command is intentionally conservative: fixture IDs cannot contain path
separators, synthetic fixture file paths must be relative, existing fixtures
are not overwritten unless `--force` is explicit, and expected rule counts are
verified before any fixture is written.
