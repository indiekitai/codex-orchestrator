# Post-Merge Docs Drift Guard

Task ID: TF-CODEX-ORCH-V4-POST-MERGE-DOCS-DRIFT-GUARD-LOCAL

## Summary

Added a bounded local/static guard to `run-routine docs-drift-checker`.
The checker now scans `docs/reviews/*.md` for accepted or merged task notes
that mention central-impacting command, routine, or source changes without
recording a central docs update or explicit docs-drift decision.

## Local Evidence

- `cmd/codex-orchestrator/main.go`: post-merge docs drift guard scan added to
  the existing docs drift checker routine.
- `cmd/codex-orchestrator/main_test.go`: added passing and failing local
  fixture coverage for review notes with and without a docs-drift decision.
- `routines/docs-drift-checker.json`, `README.md`, `README.zh-CN.md`,
  `SKILL.md`, `docs/routines/README.md`, `docs/v2-usage.md`, and
  `docs/roadmap.md`: central docs updated to describe the new guard.

## Evidence Labels

- `local`: source, routine spec, tests, and documentation were inspected and
  updated in this worktree.
- `blocked`: no runtime, device, production, release, deployment, or external
  docs publication proof was attempted or claimed.

## Boundaries

The guard is read-only at routine runtime. It does not mutate git, the ledger,
worktrees, sessions, releases, package-manager distribution, or external
systems. It reports local/static warnings only.

## Docs Drift Decision

Central docs were updated in the same branch because this accepted routine
behavior changes the documented docs-drift checker surface.
