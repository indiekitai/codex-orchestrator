# Rules Propose Review-Only MVP

Scope: conservative V4 `codex-orchestrator rules propose` MVP.

## Outcome

Added `codex-orchestrator rules propose` as a review-only command that turns
local text evidence into a proposed rule update report.

Supported inputs:

- `--from-review PATH`
- `--text TEXT`
- `--text-file PATH`

Supported outputs:

- default Markdown-like text report;
- `--json` JSON report;
- optional `--write-report PATH` JSON report file.

The command does not edit `SKILL.md`, README files, AGENTS/CLAUDE instructions,
policy files, or project rules. Report output marks every proposal with
`needsHumanReview: true` and evidence label `local`.

## Evidence Labels

- `local`: generated proposals from local text/review input using existing OPA
  policy heuristics.
- `blocked`: returned when no local input is supplied, multiple input sources
  are supplied, unreadable files are supplied, or the input is too short to
  support a reviewable proposal.

No `direct` runtime evidence is claimed.

## Verification

- `go test ./...` passed.
- `go run ./cmd/codex-orchestrator rules propose --text ... --json` passed
  with a two-paragraph local sample that intentionally exercised `OPA001` and
  `OPA005`, producing two review-only proposals.
- `go run ./cmd/codex-orchestrator policy check --repo . --json` passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --json`
  passed.

## Self-Review

- Reread the diff after implementation.
- Changed paths stayed inside the requested allowed set:
  `cmd/codex-orchestrator/**`, `README.md`, `README.zh-CN.md`, `SKILL.md`,
  `docs/roadmap.md`, and this review file.
- No command path writes live rules; only `--write-report` writes a JSON report.
- No ledger schema changes were made.
- No package-manager, release, tag, push, merge, worktree create/delete, Paseo,
  or subagent work was performed.

Residual risk: proposal quality is intentionally heuristic and conservative.
The command reuses existing OPA text heuristics where possible and otherwise
emits a generic review-only proposal; human review remains required before any
rule update is accepted.
