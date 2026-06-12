# Roadmap scorer v2 package-lane review

Date: 2026-06-12

## Scope

This change tightens `codex-orchestrator roadmap score` so it supports the next
product direction: advancing feature packages to closure instead of filling
available slots with unrelated safe tasks.

## Changes

- Default roadmap score sources now stay on explicit project planning surfaces:
  `docs/roadmap.md`, `PROGRESS.md`, and
  `docs/TastyFuture-整体开发计划与进度.md`.
- `docs/reviews/*.md` is no longer scanned by default. Review docs can still be
  included with `--config` when a project intentionally wants a selected review
  note to provide planning input.
- Review/postmortem risk sections such as `Residual Risks` no longer become
  default dispatch candidates.
- Colon-style section labels, such as `下一阶段优先级：` and
  `暂不进入的方向：`, are treated as section labels, which matches common Chinese
  planning docs.
- Feature package / package status / package lane candidates are scored above
  unrelated safe task fillers.
- The roadmap now declares the next product sequence as feature packages:
  roadmap scorer v2, package ledger/status, human-friendly status page,
  watchdog status polish, and v0.3.3 release closeout.

## Verification

- `go test ./cmd/codex-orchestrator -run 'TestRoadmapScore' -count=1`: passed.
- `go test ./...`: passed.
- `git diff --check`: passed.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`:
  passed.
- `go run ./cmd/codex-orchestrator policy check --repo .`: passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator roadmap score --repo . --json`: passed; top
  candidate is now `Package ledger / package status：待做` with action
  `dispatch as one feature-package lane, not as unrelated task filler`.

## Evidence labels

- `local`: Go tests, local/static helper output, docs/routine/policy scans.
- `proxy`: none.
- `direct`: none; this is not live Codex App session proof.
- `blocked`: none for this local/static slice.

## Residual risk

The scorer still returns lower-ranked legacy roadmap candidates from old roadmap
sections. That is acceptable for this slice because the first dispatch decision
now favors the package lane. A future package-status slice should make package
state a first-class ledger concept instead of relying only on scoring text.
