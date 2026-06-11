# Static Status Page Review

Task: `TF-CODEX-ORCH-V4-STATIC-STATUS-PAGE-LOCAL`

Commit reviewed: `286791e` (`Add static HTML status output`)

## Scope

This slice adds `codex-orchestrator status --html`, a local/static HTML status
page rendered to stdout from the existing ledger/observe data. It does not add
a server, daemon, scheduler, package-manager distribution path, automatic
merge/push/cleanup, or runtime monitor.

## Review Summary

- Changed files stayed within the approved scope:
  - `cmd/codex-orchestrator/main.go`
  - `cmd/codex-orchestrator/main_test.go`
  - `README.md`
  - `README.zh-CN.md`
- Forbidden paths were not touched:
  - no `.github/workflows/**`
  - no release scripts/assets
  - no Homebrew/npm/tap/package-manager files
- The status page includes human-readable sections for integration cleanliness,
  queue pressure, next action, evidence labels, active/pending/dirty/review/
  blocked/cleanup/recent tasks, job rows, and recent routine runs.
- Raw task ids remain available in metadata, while titles and grouped sections
  make the queue easier to scan.

## Evidence

- `local`: `go test ./cmd/codex-orchestrator -run 'TestObserveRuntimeStatusReportCategories|TestObserveJobSummaryAndProjectMap' -count=1`
- `local`: `go build ./cmd/codex-orchestrator`
- `local`: `go test ./...`
- `local`: `git diff --check f29e6809fe6c52a5f64a47e2617a4fabd56379d2..HEAD`
- `local`: `status --html` smoke generated a 40KB HTML page from the main
  repository ledger and included the expected Chinese sections and
  `local/static` boundary statement.
- `local`: `docs-drift-checker` passed.
- `local`: `evidence-label-auditor` passed with no rule hits.
- `local`: `policy check` passed with 19 orchestration policy eval fixtures.

## Boundaries

- This is local/static status evidence only.
- It does not prove Codex App runtime behavior, heartbeat delivery, daemon
  behavior, production behavior, payment behavior, hardware/device behavior, or
  worker correctness beyond ledger/git/worktree facts.
- `status --html` is a visibility surface. It does not mutate git, update the
  ledger, schedule sessions, enforce budgets, or perform cleanup.

## Residual Risks

- The page currently renders a full job list; very large ledgers may need
  filtering or grouping in a later slice.
- The page depends on the existing ledger/observe data quality. It makes
  state visible but does not by itself fix incomplete dispatch records.
- The page is static HTML only. It is intentionally not a live dashboard.

