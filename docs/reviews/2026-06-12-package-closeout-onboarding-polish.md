# Package Closeout / Onboarding Polish Review

Date: 2026-06-12

## Scope

This review records a local/static codex-orchestrator polish pass focused on:

- hiding legacy terminal ungrouped tasks from current-action status rows;
- adding starter project onboarding templates;
- adding package closeout status reporting;
- updating README / skill / roadmap documentation.

No Homebrew, npm, tap, package-manager, daemon, or runtime UI work is included.

## Evidence Label

`local/static`

The changes are helper, documentation, and test coverage changes. They do not
create Codex App sessions, dispatch workers, merge project branches, deploy,
touch external providers, or prove direct runtime/device behavior.

## Changes Reviewed

- `JobSummary` now separates:
  - full ledger rows;
  - visible current-action rows;
  - legacy terminal ungrouped rows;
  - non-terminal ungrouped task count.
- `PackageLaneGuard` no longer warns when the only ungrouped tasks are old
  terminal history.
- `status` HTML/Markdown explains hidden legacy terminal rows.
- `init --write-templates` writes starter local planning files for:
  - orchestration policy;
  - package plan;
  - project map.
- `pack status --package-id PKG` embeds package acceptance and package summary
  into a package closeout report.
- README, Chinese README, `SKILL.md`, `docs/v2-usage.md`, and roadmap docs now
  describe these flows.

## Verification

Run:

```bash
go test ./...
go run ./cmd/codex-orchestrator status --repo . --json
```

The status smoke confirmed that the repository's existing legacy ungrouped
terminal tasks no longer trigger a package-lane warning.

## Pi Review

Pi was run read-only against the local diff as proxy/advisory review evidence.

Verdict: PASS, no blocking correctness issues.

Non-blocking note applied: when all task rows are hidden legacy terminal rows,
the HTML jobs section now says "No current-action tasks" instead of "No tasks
recorded."

## Residual Risks

- `pack status` is intentionally conservative and still depends on
  `pack acceptance`; packages whose worker worktrees have already been cleaned
  may need recorded closeout artifacts rather than fresh merge-readiness packs.
- The starter templates are generic and project owners should still adapt them
  to local source-of-truth docs, gates, and forbidden paths.
- Legacy row hiding is presentation-level; full rows remain in JSON for audit.
