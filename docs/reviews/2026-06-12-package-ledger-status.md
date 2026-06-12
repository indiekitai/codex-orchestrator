# Package ledger status

Date: 2026-06-12

## Scope

This package adds first-class package lane visibility to the local helper.
It is intended to keep several related worker tasks grouped under the same
feature package instead of letting the orchestrator report only scattered task
rows.

Changed surfaces:

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `README.md`
- `README.zh-CN.md`
- `docs/v2-usage.md`
- `docs/roadmap.md`

## Implementation

- Added optional `Task.packageId` and `--package-id` support to `record-task`
  and `dispatch record`.
- Added package IDs to runtime status items and job summary rows.
- Added `packageSummary` to `observe` and `status` JSON output.
- Added package-level HTML, Markdown, and terminal rendering for active,
  review-needed, blocked, cleanup-needed, attention-needed, review-only, and
  cleaned package states.
- Added shell completion entries for `--package-id` on task recording paths.
- Updated README and usage docs so users know where package status appears.
- Updated the roadmap to mark this package as locally implemented.

## Pi review

Reviewer: Pi CLI
Evidence label: proxy/advisory

Pi reported one actionable issue: package tasks with unrecognized statuses could
fall through to `cleaned`. This was fixed by adding `otherTaskIds` and an
`attention-needed` package status, plus a regression assertion for rejected or
otherwise unrecognized task states.

Pi also noted:

- `RoutineRun.PackageID` / `RoutineRun.At` might be missing. Local compile and
  `go test ./...` disproved this for the current tree.
- String timestamp comparison assumes sortable ISO-style timestamps. Current
  ledger/routine timestamps are stored in that format; this remains a low-risk
  implementation assumption.
- A package can be `review-needed` while other workers in the same package are
  still active. The action copy was adjusted to mention continued monitoring of
  active same-lane workers.
- Routine-run-only package rows have limited test coverage. Current behavior is
  intentionally `review-only` local/static state.

External review output does not authorize merge/push/cleanup by itself.

## Verification

- `go test ./cmd/codex-orchestrator -run 'TestStatusIncludesPackageSummary|TestObserveRuntimeStatusReportCategories|TestPendingWorktreeIDTaskLifecycle' -count=1`
- `go test ./cmd/codex-orchestrator -run 'TestStatusIncludesPackageSummary|TestObserveJobSummaryAndProjectMap' -count=1`
- `go test ./...`
- `git diff --check`
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`

All gates passed.

## Evidence labels

- local/static: helper state model, tests, docs, status rendering, routine and
  policy checks.
- proxy/advisory: Pi review output.
- direct: none. This package does not create Codex App sessions or prove live
  runtime orchestration behavior.
- blocked: none.

## Residual risk

This is a local helper/status package. It improves what the orchestrator can see
and report, but it still does not dispatch workers, enforce package lane choices,
merge, push, cleanup, or run as a daemon.
