# Human-friendly status glance

Date: 2026-06-12

## Scope

This package improves the fixed status page and Markdown summary so a human can
quickly see where the orchestrator is, without reading raw JSON or long task
tables first.

Changed surfaces:

- `cmd/codex-orchestrator/main.go`
- `cmd/codex-orchestrator/main_test.go`
- `README.md`
- `README.zh-CN.md`
- `docs/v2-usage.md`
- `docs/roadmap.md`

## Implementation

- Added an "一眼看懂 / At a Glance" section to HTML status output.
- Added the same section near the top of Markdown summaries.
- The summary highlights integration cleanliness, missed heartbeat signals,
  current package lane, review/blocker/cleanup pressure, dispatch slot pressure,
  and the first recommended action.
- Kept the detailed runtime/package/job sections below the human summary.
- Updated README and usage docs to describe the new status surface.
- Updated the roadmap to mark the first local status-glance slice complete.

## Pi review

Reviewer: Pi CLI
Evidence label: proxy/advisory

Pi found no blocking bugs. It pointed out three polish items:

- branch coverage was narrow;
- "repo truth" was jargon in user-facing copy;
- README said the HTML status starts with At a Glance while the section was
  below the metric cards.

Follow-up changes:

- Added focused coverage for dirty integration, missed heartbeat, active/pending
  work, full dispatch slots, and absent recommendations.
- Replaced the user-facing "repo truth" wording with "状态刷新".
- Moved At a Glance ahead of the HTML metric grid and ahead of the Markdown
  machine summary.

External review output does not authorize merge/push/cleanup by itself.

## Verification

- `go test ./cmd/codex-orchestrator -run 'TestStatusIncludesPackageSummary|TestStatusAtAGlanceLinesCoverAttentionBranches|TestObserveJobSummaryAndProjectMap' -count=1`
- `go test ./...`
- `git diff --check`

Additional project gates are run before merge.

## Evidence labels

- local/static: status rendering, Markdown summary, tests, docs.
- proxy/advisory: Pi review output.
- direct: none. This package does not create Codex App sessions or prove live
  runtime orchestration behavior.
- blocked: none.

## Residual risk

The status glance is intentionally a concise local/static summary. It improves
visibility for humans, but does not enforce task choice, keep the Mac awake,
dispatch sessions, merge, push, cleanup, or replace detailed review evidence.
