# Misalignment Reduction Alpha Review

Date: 2026-06-13

## Scope

Implemented the first local/static developer-agent misalignment reduction loop:

- ledger-backed misalignment events;
- constraint-stack snapshots on task records;
- evidence-bound claim verification in merge-readiness and acceptance reports;
- trust-risk status output;
- `OPA010` policy/eval coverage for unsupported completion claims;
- English/Chinese documentation and release-note updates.

## Changed Areas

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `eval/orchestration-policy-auditor/claim-self-report-without-verification.json`
- `README.md`
- `README.zh-CN.md`
- `SKILL.md`
- `docs/full-guide.md`
- `docs/full-guide.zh-CN.md`
- `docs/roadmap.md`
- `docs/routines/README.md`
- `docs/research/developer-agent-misalignment.md`
- `docs/beta-release-notes-draft.md`
- `docs/distribution-package.md`

## Evidence Boundary

All new output is `local/static` orchestration evidence. The new trust-risk,
misalignment, and claim-verification reports do not prove model intent,
production behavior, live provider state, device behavior, Codex App heartbeat
delivery, or direct runtime proof.

`claimVerification` can block acceptance or require human review, but it is not
automatic authorization to merge, push, release, deploy, clean a worktree, or
claim a feature package is complete.

## Verification Plan

Required before release:

- `go test ./...`
- `go run ./cmd/codex-orchestrator eval run --json`
- `go run ./cmd/codex-orchestrator policy check --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine orchestration-policy-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator help`
- `git diff --check`

Optional but preferred:

- Pi review on the implementation diff, if the local Pi review command is
  available.

## Self-Review Notes

- No Homebrew/npm/tap/package-manager distribution work is included.
- The public restaurant POS case remains generic; no private project name was
  reintroduced.
- The new policy fixture is deterministic and does not depend on private
  transcripts.
- The new misalignment report records local events and recommendations only; it
  does not upload or mine user chats.
- The status-page trust-risk block is advisory and should not be treated as a
  daemon/runtime proof.
