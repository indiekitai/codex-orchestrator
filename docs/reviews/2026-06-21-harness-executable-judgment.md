# Harness executable judgment update

Date: 2026-06-21

## Input

- Public X post: <https://x.com/ma_zhenyuan/status/2066135520586138039>

## Summary

The post framed Harness as more than tools, MCP, knowledge bases, skills, or
schemas. The useful product lesson for `codex-orchestrator` is that a harness
must encode the project's own definition of "what counts as correct" into
source-of-truth context, tool/path boundaries, output shape, gates, evidence
labels, review feedback, and stop conditions.

This fits the existing positioning, so the main product tagline was not
changed. The update narrows the explanation of "engineering harness" and makes
worker contracts explicitly carry acceptance definitions and source-of-truth
context.

## Changes

- `README.md`: added a short "What Harness Means Here" section.
- `README.zh-CN.md`: added the corresponding Chinese section.
- `SKILL.md`: clarified Harness as executable project judgment.
- `SKILL.md`: added `Acceptance definition` and `Source-of-truth context` to
  delegated worker prompt contracts.
- `SKILL.md`: added a worker prompt guard against replacing source truth with a
  keyword summary, generic pattern, or invented checklist.
- `docs/research/harness-reading-notes.md`: added a later note on Harness as
  executable judgment.

## Evidence Labels

- `proxy`: the public X post is product-framing input, not product proof.
- `local`: repository docs and skill instructions were updated locally.
- `local`: verification commands below passed.

## Verification

- `git diff --check`
- `go test ./...`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --write-report /tmp/codex-orchestrator-docs-drift.json --json`
- `go run ./cmd/codex-orchestrator eval run --write-report /tmp/codex-orchestrator-eval-run.json --json`
- `codex --help`
- `codex-orchestrator --help`

## Residual Risk

This is a docs/skill rule update only. It does not add a new helper gate that
machine-checks every future worker prompt for `Acceptance definition` or
`Source-of-truth context`; that could become a future policy/eval fixture if
real runs show workers still inventing project rules.
